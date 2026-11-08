package bridge

import (
	"bytes"
	"io"
	"regexp"
	"strings"
)

const maxLocalizedLine = 16 << 10

var boxCapacityPattern = regexp.MustCompile(
	`^box rsh: box (.+?) failed to start: node at capacity, try again shortly(\r?\n)?$`,
)

// localizeBoxOutput translates the startup messages users see in the bridge.
// It holds only an incomplete possible match, so a shell prompt is not delayed.
func localizeBoxOutput(dst io.Writer, src io.Reader) error {
	localizer := &boxLocalizer{dst: dst}
	buf := make([]byte, 32<<10)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			if writeErr := localizer.Write(buf[:n]); writeErr != nil {
				return writeErr
			}
		}
		if err != nil {
			if err == io.EOF {
				return localizer.Flush()
			}
			return err
		}
	}
}

type boxLocalizer struct {
	dst     io.Writer
	pending []byte
}

func (l *boxLocalizer) Write(p []byte) error {
	l.pending = append(l.pending, p...)
	for {
		if i := bytes.IndexByte(l.pending, '\n'); i >= 0 {
			if err := l.writeLine(l.pending[:i+1]); err != nil {
				return err
			}
			l.pending = l.pending[i+1:]
			continue
		}
		if translated, ok := localizeCapacityLine(l.pending); ok {
			if _, err := l.dst.Write(translated); err != nil {
				return err
			}
			l.pending = nil
			return nil
		}
		if !couldStartBoxMessage(l.pending) || len(l.pending) >= maxLocalizedLine {
			return l.flushPending()
		}
		return nil
	}
}

func (l *boxLocalizer) Flush() error {
	return l.flushPending()
}

func (l *boxLocalizer) writeLine(line []byte) error {
	if translated, ok := localizeBoxLine(line); ok {
		_, err := l.dst.Write(translated)
		return err
	}
	_, err := l.dst.Write(line)
	return err
}

func (l *boxLocalizer) flushPending() error {
	if len(l.pending) == 0 {
		return nil
	}
	_, err := l.dst.Write(l.pending)
	l.pending = nil
	return err
}

func localizeBoxLine(line []byte) ([]byte, bool) {
	const waking = "box: waking "
	if bytes.HasPrefix(line, []byte(waking)) {
		body := line[len(waking):]
		var newline []byte
		switch {
		case bytes.HasSuffix(body, []byte("\r\n")):
			body, newline = body[:len(body)-2], []byte("\r\n")
		case bytes.HasSuffix(body, []byte("\n")):
			body, newline = body[:len(body)-1], []byte("\n")
		}
		return appendBoxLine("box：正在唤醒 "+string(body)+"。", newline), true
	}
	return localizeCapacityLine(line)
}

func localizeCapacityLine(line []byte) ([]byte, bool) {
	if match := boxCapacityPattern.FindSubmatch(line); match != nil {
		message := "box rsh：Box " + string(match[1]) + " 启动失败：云端节点资源已满，请稍后重试。"
		return appendBoxLine(message, match[2]), true
	}
	return nil, false
}

func appendBoxLine(message string, newline []byte) []byte {
	if len(newline) == 0 {
		return []byte(message)
	}
	return append([]byte(message), newline...)
}

func couldStartBoxMessage(p []byte) bool {
	if len(p) == 0 {
		return false
	}
	for _, prefix := range []string{"box: waking ", "box rsh: box "} {
		if strings.HasPrefix(prefix, string(p)) || bytes.HasPrefix(p, []byte(prefix)) {
			return true
		}
	}
	return false
}
