## Building

Create a dir somewhere and then run the following:

`export GOPATH=$(pwd)`

`mkdir -p src/github.com/iguazio && cd src/github.com/iguazio`

`git clone git@github.com:iguazio/kibini.git && cd kibini`

`make`

The kibini binary should be in $GOPATH/bin for both OSX and Linux64

## Using
`kibini export --help`

`kibini import --help`
