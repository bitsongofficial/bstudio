_NOTE: This is alpha software. Please contact us if you aim to run it in production._

**Note**: Requires [Go 1.13.6+](https://golang.org/dl/)

# Install BStudio

## From Source
1. **Install Go** by following the [official docs](https://golang.org/doc/install). Remember to set your `$GOPATH` and `$PATH` environment variables, for example:
    ```bash
    wget https://dl.google.com/go/go1.13.6.linux-amd64.tar.gz
    sudo tar -xvzf go1.13.6.linux-amd64.tar.gz
    sudo mv go /usr/local
     
    cat <<EOF >> ~/.profile  
    export GOPATH=$HOME/go  
    export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin  
    EOF
    ```
2. **Clone BStudio source code to your machine**
    ```bash
    mkdir -p $GOPATH/src/github.com/bitsongofficial
    cd $GOPATH/src/github.com/bitsongofficial
    git clone https://github.com/bitsongofficial/bstudio.git
    cd bstudio
    ```
3. **Compile**
    ```bash
    # Install the app into your $GOBIN
    make all
    # Now you should be able to run the following commands:
    bstudio help
    ```
    The latest `bstudio version` is now installed.
3. **Run BStudio**
	```bash
	bstudio start
	```
5. [Test with Swagger](http://localhost:1347/swagger/index.html)
