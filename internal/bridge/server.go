package bridge

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/binary"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/creack/pty"
	"golang.org/x/crypto/ssh"
)

func Run(version string, args []string) error {
	cfg, showVersion, err := parseFlags(args)
	if err != nil {
		return err
	}
	if showVersion {
		fmt.Println(version)
		return nil
	}
	if cfg.BoxCLI == "" {
		cfg.BoxCLI = DiscoverBoxCLI()
	}
	if cfg.BoxCLI == "" {
		return errors.New("cannot find @simular-ai/box; pass --box-cli")
	}
	if len(cfg.BoxUsers) == 0 {
		cfg.BoxUsers = map[string]BoxRoute{cfg.User: {Box: cfg.Box}}
	}
	if cfg.Password == "" && cfg.AuthorizedKey == "" {
		cfg.Password = randomToken(18)
		fmt.Fprintf(os.Stderr, "SSH password: %s\n", cfg.Password)
	}

	signer, err := LoadOrCreateHostKey(cfg.HostKey)
	if err != nil {
		return fmt.Errorf("host key: %w", err)
	}

	server := &ssh.ServerConfig{}
	if cfg.AuthorizedKey != "" {
		keys, err := LoadAuthorizedKeys(cfg.AuthorizedKey)
		if err != nil {
			return fmt.Errorf("authorized key: %w", err)
		}
		server.PublicKeyCallback = func(metadata ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if _, ok := cfg.boxForUser(metadata.User()); ok && keys.Contains(key) {
				return nil, nil
			}
			return nil, errors.New("authentication failed")
		}
	}
	if cfg.Password != "" {
		server.PasswordCallback = func(metadata ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			if _, ok := cfg.boxForUser(metadata.User()); ok && constantTimeEqual(string(password), cfg.Password) {
				return nil, nil
			}
			return nil, errors.New("authentication failed")
		}
	}
	server.AddHostKey(signer)

	listener, err := net.Listen("tcp", cfg.Listen)
	if err != nil {
		return err
	}
	defer listener.Close()
	log.Printf("SSH bridge listening on %s", cfg.String())

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)
	go func() {
		<-signals
		_ = listener.Close()
	}()

	for {
		raw, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			log.Printf("accept: %v", err)
			continue
		}
		go handleConn(raw, server, cfg)
	}
}

func parseFlags(args []string) (Config, bool, error) {
	cfg := Config{StateDir: DefaultStateDir()}
	flags := flag.NewFlagSet("box-ssh-bridge", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	var version bool
	flags.StringVar(&cfg.Listen, "listen", defaultListen, "local SSH listen address")
	flags.StringVar(&cfg.Box, "box", defaultBox, "Simular Box name or ID")
	flags.StringVar(&cfg.User, "user", defaultUser, "SSH username exposed to clients")
	flags.StringVar(&cfg.UsersFile, "users-file", "", "JSON map of SSH username to Box name or ID")
	flags.StringVar(&cfg.Password, "password", "", "SSH password; one is generated when no key authentication is configured")
	flags.StringVar(&cfg.AuthorizedKey, "authorized-key", "", "authorized_keys file; defaults to $BOX_SSH_BRIDGE_HOME/authorized_keys when present")
	flags.StringVar(&cfg.HostKey, "host-key", "", "SSH host key path")
	flags.StringVar(&cfg.BoxCLI, "box-cli", "", "path to simular-box CLI entrypoint")
	flags.StringVar(&cfg.BoxPackage, "box-package", "@simular-ai/box", "package used when --box-cli is npx")
	flags.StringVar(&cfg.SFTPCommand, "sftp-command", "", "remote command that serves an SFTP subsystem")
	flags.StringVar(&cfg.StateDir, "state-dir", cfg.StateDir, "state directory")
	flags.BoolVar(&version, "version", false, "print version")
	if err := flags.Parse(args); err != nil {
		return Config{}, false, err
	}
	if cfg.HostKey == "" {
		cfg.HostKey = defaultHostKey(cfg.StateDir)
	}
	if cfg.AuthorizedKey == "" {
		path := defaultAuthorizedKeys(cfg.StateDir)
		if _, err := os.Stat(path); err == nil {
			cfg.AuthorizedKey = path
		}
	}
	if cfg.UsersFile != "" {
		users, err := LoadUsers(cfg.UsersFile)
		if err != nil {
			return Config{}, false, err
		}
		cfg.BoxUsers = users
	}
	if cfg.BoxUsers == nil && !configHasFlag(args, "box") && !configHasFlag(args, "user") {
		users, err := LoadUsers(filepath.Join(cfg.StateDir, "users.json"))
		if err == nil {
			cfg.BoxUsers = users
		}
	}
	return cfg, version, nil
}

func handleConn(raw net.Conn, server *ssh.ServerConfig, cfg Config) {
	defer raw.Close()
	_ = raw.SetDeadline(time.Now().Add(30 * time.Second))
	conn, channels, requests, err := ssh.NewServerConn(raw, server)
	if err != nil {
		log.Printf("handshake from %s: %v", raw.RemoteAddr(), err)
		return
	}
	_ = raw.SetDeadline(time.Time{})
	defer conn.Close()
	log.Printf("SSH session from %s", raw.RemoteAddr())
	go ssh.DiscardRequests(requests)
	for incoming := range channels {
		if incoming.ChannelType() != "session" {
			_ = incoming.Reject(ssh.UnknownChannelType, "only session channels are supported")
			continue
		}
		channel, channelRequests, err := incoming.Accept()
		if err != nil {
			continue
		}
		route, ok := cfg.boxForUser(conn.User())
		if !ok {
			_ = channel.Close()
			continue
		}
		go handleSession(channel, channelRequests, cfg, route)
	}
}

type session struct {
	channel ssh.Channel
	cfg     Config
	route   BoxRoute
	term    string
	rows    uint32
	cols    uint32
	pty     bool
	started bool
	closed  chan struct{}
	master  *os.File
}

func handleSession(channel ssh.Channel, requests <-chan *ssh.Request, cfg Config, route BoxRoute) {
	defer channel.Close()
	s := &session{channel: channel, cfg: cfg, route: route, term: "xterm-256color", rows: 24, cols: 80, closed: make(chan struct{})}
	for request := range requests {
		if !handleRequest(s, request) {
			return
		}
	}
}

func handleRequest(s *session, request *ssh.Request) bool {
	switch request.Type {
	case "subsystem":
		var payload struct{ Name string }
		if err := ssh.Unmarshal(request.Payload, &payload); err != nil || payload.Name != "sftp" || s.cfg.SFTPCommand == "" || s.started {
			_ = request.Reply(false, nil)
			return true
		}
		s.started = true
		_ = request.Reply(true, nil)
		go s.runLocal(s.cfg.SFTPCommand)
	case "pty-req":
		term, rows, cols, err := ParsePTY(request.Payload)
		if err == nil {
			s.term, s.rows, s.cols, s.pty = term, rows, cols, true
		}
		_ = request.Reply(err == nil, nil)
	case "window-change":
		rows, cols, err := ParseWindowChange(request.Payload)
		if err == nil {
			s.rows, s.cols = rows, cols
			if s.master != nil {
				_ = pty.Setsize(s.master, &pty.Winsize{Rows: uint16(rows), Cols: uint16(cols)})
			}
		}
	case "env":
		_ = request.Reply(true, nil)
	case "shell", "exec":
		if s.started {
			_ = request.Reply(false, nil)
			return true
		}
		command := "bash -li"
		if request.Type == "exec" {
			var payload struct{ Command string }
			if err := ssh.Unmarshal(request.Payload, &payload); err != nil || payload.Command == "" {
				_ = request.Reply(false, nil)
				return true
			}
			command = payload.Command
		}
		s.started = true
		_ = request.Reply(true, nil)
		go s.run(command)
	default:
		_ = request.Reply(false, nil)
	}
	return true
}

func (s *session) run(command string) {
	cmd := boxRSHCommand(s.cfg, s.route.Box, command, s.pty)
	cmd.Env = append(os.Environ(), "TERM="+s.term, "COLUMNS="+fmt.Sprint(s.cols), "LINES="+fmt.Sprint(s.rows))
	if s.route.BoxHome != "" {
		cmd.Env = append(cmd.Env, "BOX_HOME="+s.route.BoxHome)
	}

	input, output, errOutput, err := startCommand(cmd, s)
	if err != nil {
		s.exit(255, err.Error())
		return
	}
	go func() {
		if s.pty {
			_ = localizeBoxOutput(s.channel, output)
			return
		}
		_, _ = io.Copy(s.channel, output)
	}()
	if errOutput != nil {
		go func() {
			_ = localizeBoxOutput(s.channel.Stderr(), errOutput)
		}()
	}
	go func() {
		_, _ = io.Copy(input, s.channel)
		_ = input.Close()
	}()
	if s.pty {
		go func() {
			<-s.closed
			_ = input.Close()
		}()
	}

	waitErr := cmd.Wait()
	code := 0
	if waitErr != nil {
		var exitErr *exec.ExitError
		if errors.As(waitErr, &exitErr) {
			code = exitErr.ExitCode()
		} else {
			code = 255
		}
	}
	s.exit(uint32(code), "")
}

func (s *session) runLocal(command string) {
	command = strings.ReplaceAll(command, "{box}", shellQuote(s.route.Box))
	command = strings.ReplaceAll(command, "{box_home}", shellQuote(s.route.BoxHome))
	cmd := exec.Command("sh", "-c", command)
	input, output, errOutput, err := startCommand(cmd, s)
	if err != nil {
		s.exit(255, err.Error())
		return
	}
	go func() {
		_, _ = io.Copy(s.channel, output)
	}()
	if errOutput != nil {
		go func() {
			_, _ = io.Copy(s.channel.Stderr(), errOutput)
		}()
	}
	go func() {
		_, _ = io.Copy(input, s.channel)
		_ = input.Close()
	}()
	waitErr := cmd.Wait()
	code := 0
	if waitErr != nil {
		var exitErr *exec.ExitError
		if errors.As(waitErr, &exitErr) {
			code = exitErr.ExitCode()
		} else {
			code = 255
		}
	}
	s.exit(uint32(code), "")
}

func startCommand(cmd *exec.Cmd, s *session) (io.WriteCloser, io.ReadCloser, io.ReadCloser, error) {
	if s.pty {
		tty, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: uint16(s.rows), Cols: uint16(s.cols)})
		if err != nil {
			return nil, nil, nil, err
		}
		s.master = tty
		return tty, tty, nil, nil
	}
	input, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, nil, err
	}
	output, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, nil, err
	}
	errOutput, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, nil, nil, err
	}
	return input, output, errOutput, nil
}

func (s *session) exit(code uint32, message string) {
	select {
	case <-s.closed:
		return
	default:
		close(s.closed)
	}
	if message != "" {
		_, _ = io.WriteString(s.channel.Stderr(), message+"\r\n")
	}
	payload := make([]byte, 4)
	binary.BigEndian.PutUint32(payload, code)
	_, _ = s.channel.SendRequest("exit-status", false, payload)
	_ = s.channel.CloseWrite()
	_ = s.channel.Close()
}

type authorizedKeys map[string]bool

func (a authorizedKeys) Contains(key ssh.PublicKey) bool {
	return a[string(key.Marshal())]
}

func LoadAuthorizedKeys(path string) (authorizedKeys, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	keys := authorizedKeys{}
	for len(data) > 0 {
		key, _, _, rest, err := ssh.ParseAuthorizedKey(data)
		if err != nil {
			return nil, err
		}
		keys[string(key.Marshal())] = true
		data = rest
	}
	return keys, nil
}

func LoadOrCreateHostKey(path string) (ssh.Signer, error) {
	if data, err := os.ReadFile(path); err == nil {
		return ssh.ParsePrivateKey(data)
	}
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	block, err := ssh.MarshalPrivateKey(privateKey, "")
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, pem.EncodeToMemory(block), 0600); err != nil {
		return nil, err
	}
	return ssh.NewSignerFromKey(privateKey)
}

func randomToken(n int) string {
	const alphabet = "abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	for i := range buf {
		buf[i] = alphabet[int(buf[i])%len(alphabet)]
	}
	return string(buf)
}

func constantTimeEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var diff byte
	for i := range a {
		diff |= a[i] ^ b[i]
	}
	return diff == 0
}

func DiscoverBoxCLI() string {
	home, _ := os.UserHomeDir()
	candidates := []string{}
	if runtime.GOOS == "windows" {
		candidates = append(candidates, filepath.Join(home, "AppData", "Roaming", "npm", "simular-box.cmd"))
	} else {
		candidates = append(candidates, filepath.Join(home, ".npm-global", "bin", "simular-box"), "/usr/local/bin/simular-box", "/opt/homebrew/bin/simular-box")
		matches, _ := filepath.Glob(filepath.Join(home, ".npm", "_npx", "*", "node_modules", "@simular-ai", "box", "bin", "simular-box.js"))
		candidates = append(candidates, matches...)
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return ""
}
