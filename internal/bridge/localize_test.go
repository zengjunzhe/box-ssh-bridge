package bridge

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestLocalizeBoxCapacityMessageAcrossChunks(t *testing.T) {
	input := "box: wak" +
		"ing main\r\n" +
		"box rsh: box main failed to start: node at capacity, try again short" +
		"ly\r\n" +
		"user@main:/workspace$ "
	var output bytes.Buffer
	if err := localizeBoxOutput(&output, &chunkReader{data: []byte(input), size: 7}); err != nil {
		t.Fatal(err)
	}
	got := output.String()
	want := "box：正在唤醒 main。\r\n" +
		"box rsh：Box main 启动失败：云端节点资源已满，请稍后重试。\r\n" +
		"user@main:/workspace$ "
	if got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
	if strings.Contains(got, "node at capacity") {
		t.Fatal("English capacity error was not translated")
	}
}

func TestLocalizeBoxOutputLeavesOtherMessagesAlone(t *testing.T) {
	input := "box rsh: box main unavailable\r\nuser@main:/workspace$ "
	var output bytes.Buffer
	if err := localizeBoxOutput(&output, strings.NewReader(input)); err != nil {
		t.Fatal(err)
	}
	if output.String() != input {
		t.Fatalf("output = %q, want %q", output.String(), input)
	}
}

type chunkReader struct {
	data []byte
	size int
}

func (r *chunkReader) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	n := r.size
	if n > len(r.data) {
		n = len(r.data)
	}
	if n > len(p) {
		n = len(p)
	}
	copy(p, r.data[:n])
	r.data = r.data[n:]
	return n, nil
}
