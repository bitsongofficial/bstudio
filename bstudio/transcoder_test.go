package bstudio

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestTranscoder_Ffprobe(t *testing.T) {
	bs := mockBStudio()
	tr := NewTranscoder(bs, "QmZWCE29y6omGw8vuiQQpMKehfrhggxytjCd9McxRomsLt")
	d, err := tr.GetCidDuration()
	require.NoError(t, err)
	fmt.Println(d)
}

func TestTranscoder_Transcode(t1 *testing.T) {
	bs := mockBStudio()
	_ = NewTranscoder(bs, "QmZWCE29y6omGw8vuiQQpMKehfrhggxytjCd9McxRomsLt")
	//fmt.Println(tr.Transcode())
}
