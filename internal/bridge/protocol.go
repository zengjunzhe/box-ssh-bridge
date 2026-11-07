package bridge

import "golang.org/x/crypto/ssh"

func ParsePTY(payload []byte) (string, uint32, uint32, error) {
	var request struct {
		Term   string
		Cols   uint32
		Rows   uint32
		Width  uint32
		Height uint32
		Modes  string
	}
	if err := ssh.Unmarshal(payload, &request); err != nil {
		return "", 0, 0, err
	}
	return request.Term, request.Rows, request.Cols, nil
}

func ParseWindowChange(payload []byte) (uint32, uint32, error) {
	var request struct {
		Cols   uint32
		Rows   uint32
		Width  uint32
		Height uint32
	}
	if err := ssh.Unmarshal(payload, &request); err != nil {
		return 0, 0, err
	}
	return request.Rows, request.Cols, nil
}
