package ipfs

/*import (
	"fmt"
	files "github.com/ipfs/go-ipfs-files"
	"github.com/ipfs/go-ipfs/core/coreunix"
	coreiface "github.com/ipfs/interface-go-ipfs-core"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNode(t *testing.T) {
	node, ctx, err := Start()
	if err != nil {
		fmt.Println(fmt.Errorf("error starting ipfs: %v", err))
		return
	}

	out := make(chan interface{}, 10)
	adder, err := coreunix.NewAdder(ctx, node.Pinning, node.Blockstore, node.DAG)
	if err != nil {
		t.Fatal(err)
	}
	adder.Out = out

	fileContent := "testfileA"

	file := files.NewBytesFile([]byte(fileContent))

	go func() {
		defer close(out)
		_, _ = adder.AddAllAndPin(file)
	}()

	for o := range out {
		cid := o.(*coreiface.AddEvent).Path.Cid().String()
		require.Equal(t, "QmXcKyuajqj1cWpb31Z8EhVvQZA8JKaQBnWioffhKe7dGV", cid)
	}

	bz, err := Cat(node, "QmXcKyuajqj1cWpb31Z8EhVvQZA8JKaQBnWioffhKe7dGV", 10)
	if err != nil {
		fmt.Println(fmt.Errorf("%v", err))
		return
	}

	require.Equal(t, fileContent, string(bz))
}
*/
