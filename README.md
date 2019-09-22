# GoLogAnalyzer

A Go webserver for expose log files over HTTP with a custom configuration.

## Getting Started

This tool is developed for have few HTTP API interfaces in order to query the log files, expose the content or filter particular information from them. This tool try to be (as best as i can) compliant with a REST implementation. Swap a front end application with another can be easy if have similar API.

The tool is intended to:

- Run only on Linux machine (*use of tail for filter last log lines files*);  
- Bind the necessary network resources only under "localhost" control;  
- Not expose data over not authorized network (no input validation);  
- Consume as much low memory as possible (data are compressed in memory using the Facebook's Zstandard);  
- Don't eat CPU (be inactive the most of the time);  

During the development of the source code, I'll will try as much as i can to write modular function that can be replaced or swapped for other OS.

## Prerequisites

The software is coded in Go, and use GNU tail for extract the latest lines from the logfiles
`find` command is found in `findutils`. In order to install the `find` command you can move 2 way:

### Ubuntu

```bash
$ sudo apt install findutils
```

### CentOS

### Compile from source

```bash
mkdir -p /opt/SP/packages
cd $_
wget https://ftp.gnu.org/pub/gnu/findutils/findutils-4.7.0.tar.xz
tar xf findutils-4.7.0.tar.xz
cd findutils-4.7.0
./configure --enable-leaf-optimisation --enable-d_type-optimization --enable-threads=posix --disable-assert --enable-compiler-warnings --with-packager --with-libintl-prefix="/usr/" --with-libiconv-prefix=/usr | egrep "libi|... no"
make
make install
```

### Install from yum

```bash
$ sudo yum install findutils
```

## Install Golang

In order to install golang in your machine, you have to run the following commands:

- NOTE:  
  - It's preferable __*to don't run these command as root*__. Simply *`chown`* the *`root_foolder`* of golang to be compliant with your user and run the script;  
  - Run this "installer" script only once;  

```bash
golang_version="1.13"
golang_link="https://dl.google.com/go/go$golang_version.linux-amd64.tar.gz"
root_foolder="/opt/GOLANG" # Set the tree variable needed for build the enviroinment
go_source="$root_foolder/go"
go_projects="$root_foolder/go_projects"

# Check if this script was alredy run
if [ -d "$root_foolder" ] || [ -d "$go_source" ] || [ -d "$go_projects" ]; then
  ### Take action if $DIR exists ###
  echo "Golang is alredy installed!"
  exit 1
fi
# Be sure that golang is not alredy installed
command -v go >/dev/null 2>&1 && { echo >&2 "Seems that go is alredy installed in $(which go)"; exit 2 }

mkdir -p $root_foolder # creating dir for golang source code
cd $root_foolder # entering dir
wget $golang_link #downloading golang
tar xf $(ls | grep "tar") # extract only the tar file
mkdir $go_projects

# Add Go to the current user path
echo '
export GOPATH="$go_projects"
export GOBIN="$GOPATH/bin"
export GOROOT="$go_source"
export PATH="$PATH:$GOROOT/bin:$GOBIN"
' >> /home/$(whoami)/.bashrc

# Load the fresh changed .bashrc env file
source /home/$(whoami)/.bashrc

# Print the golang version
go version
```

After running these command, you have to be able to see the golang version installed.

### Post Prerequisites

__*NOTE*__:

- *It's preferable to logout and login to the system for a fresh reload of the configuration after have installed all the packaged listed below.*  

## Installing

GoLog-Viewer use the new golang modules manager. You can retrieve the source code by the following command:

```bash
  go get -v -u github.com/alessiosavi/GoLog-Viewer
```

In case of problem, you have to download it manually

```bash
  cd $GOPATH/src
  git clone --depth=1 https://github.com/alessiosavi/GoLog-Viewer.git
  cd GoLog-Viewer
  go clean
  go build
```

## Documentation

### HELP

For print the simple documentation

- Without compile: `go run GoLog-Viewer.go --help`  
- Compiling: `go build; ./GoLog-Viewer --help`  

```text
-gcSleep int
        Number of minutes to sleep beetween every forced GC cycle (default 10)
  -host string
        Host to bind the service (default "localhost", "0.0.0.0" for don't restrict traffic to localhost)
  -lines int
        Number of (last) lines that have to be filtered from the log (default 2000)
  -maxlines int
        Max lines used while searching for the data (default 100000)
  -path string
        Log folder that we want to expose (MANDATORY PARAMETER)
  -port int
        Port to bind the service (default 80)
  -sleep int
        Seconds for wait before check a new time if logs have changed (default 5)
```

#### Example

`go build; ./GoLog-Viewer --path /var/log --port 8081`

## Running the tests

Unfortunatly no test are provided with the initial versione of the software :/

## Deployment

You can deploy the application in two methods:

- Deploy the executable in your remote machine;  
  - Build the source in your local machine;  
  - Deploy the executable;  
- Copy the code to you remote machine, run the following commands;  

```bash
scp -r GoLog-Viewer.go user@machine.dev-prod_log:/home/no_sudo_user/gologviewer #Copy the code into your user folder
ssh user@machine.dev-prod_log # Log in into the machine
cd /home/no_sudo_user/gologviewer
go build
nohup ./gologviewer -path=/opt/SP/log &
```

Now that you have a fresh version of the code and you are in the directory of the sources file

```bash
exe="GoLog-Viewer" # Name of the executable generated
code="GoLog-Viewer.go" # Name of the main source code

echo "Killing the process ..."
pkill $exe # Killing all process that are named like $exe value
echo "Deleting old code ..."
truncate -s0 $code # Empty the file containing the old code
echo "Copy your code"
vi $code # Paste the code here
echo "Cleaning old compilation files ..."
go clean # Remove build executable
echo "Copy the new utilies sources files ..."
cp -r $code utils $GOPATH/src/gologviewer # Copy the code into the $GOPATH
echo "Building new executables ... "
go build $code
echo "Stripping debug symbols"
strip -s $exe
mkdir logs # create a folder for the logs
nohup ./$exe -path utils -port 8080 > logs/logs.txt & # Just run in background
```

## Built With

- [FastHTTP](https://github.com/valyala/fasthttp) - HTTP Framework | Tuned for high performance. Zero memory allocations in hot paths. Up to 10x faster than net/http  
- [gozstd](https://github.com/valyala/gozstd) - Facebook's compress algorithm wrapper for golang | Very usefull for performance/compression ratio  
- [logrus](https://github.com/Sirupsen/logrus) - Pretty logging framework | Not the fastest but very cool and customizable  
  - [filename](https://github.com/onrik/logrus/filename) - Plugin for logrus | Used fo print the filename and the logline at each entries of the log  
- [GoUtils](https://github.com/alessiosavi/GoUtils) - A set of Go methods for enhance productivity  

## Contributing

- Feel free to open issue in order to __*require new functionality*__;  
- Feel free to open issue __*if you discover a bug*__;  
- New idea/request/concept are very appreciated!;  

## Versioning

We use [SemVer](http://semver.org/) for versioning.

## Authors

- **Alessio Savi** - *Initial work & Concept* - [IBM Client Innovation Center [CIC]](https://github.ibm.com/Alessio-Savi)  

## License

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details

## Acknowledgments

This backend tool it's intended to run over a VPN and be served "*proxypassed & secured*" by a webserver like Apache or Nginx, in order to crypt the traffic and provide a good layer of security.

However, few basic security enhancements will be developed just for fun.

__*DO NOT RUN THIS TOOL AS SUDOUSERS - ROOT*__
