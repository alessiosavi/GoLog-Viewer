// GoLog Viewer.
//
// A simple Golang tool for expose log file over HTTP.

package main

/* ------------- IMPORT ------------- */

import (
	"bytes"
	"encoding/json"
	"flag"
	"strconv"
	"strings"
	"sync"
	"time"

	fileutils "github.com/alessiosavi/GoGPUtils/files"
	stringutils "github.com/alessiosavi/GoGPUtils/string"
	"github.com/alessiosavi/GoLog-Viewer/datastructure"

	utils "github.com/alessiosavi/GoUtils"
	"github.com/onrik/logrus/filename" // Used for print the name and the logline at each entries of the log file
	log "github.com/sirupsen/logrus"   // Pretty log library, not the fastest (zerolog/zap)
	"github.com/valyala/fasthttp"      // external package used for networking
	"github.com/valyala/gozstd"        // Valyala wrapper implementation of the Facebook zstd compressing alghoritm
)

/* ------------- INIT ------------- */
func main() {
	var (
		channel        bool                          // Used for "thread safety" during "hot change" of datastructure.Configuration
		logCfg         datastructure.Configuration   // The data structure for save the datastructure.Configuration
		fileListStruct []datastructure.LogFileStruct // The data structure for save every the files log information
	)

	Formatter := new(log.TextFormatter)
	// Formatter.TimestampFormat = "15-01-2018 15:04:05.000000"
	Formatter.FullTimestamp = true
	Formatter.ForceColors = true
	log.AddHook(filename.NewHook()) // Print filename + line at every log
	log.SetFormatter(Formatter)
	log.SetLevel(log.DebugLevel)

	logCfg = InitConfigurationData()          // Init the datastructure.Configuration
	fileListStruct = InitLogFileData(&logCfg) // Initialize the data
	channel = true
	go CoreEngine(fileListStruct, &logCfg, &channel)  // Run the core engine as a background task
	HandleRequests(fileListStruct, &logCfg, &channel) // Spawn the HTTP service for serve the request
}

/* ------------- CORE METHOD ------------- */

// CoreEngine Main core function for recognize file change. It have to scan the list of file recognize if a file have changed.
// In order to achieve an higher efficiency and be compliant with every SO this function is developed in pure GO.
// The datastructure.Configuration of the tool can change at runtime using the API.A boolean channel it's used for be sure to read the data accordly to the latest datastructure.Configuration.
func CoreEngine(fileList []datastructure.LogFileStruct, logCfg *datastructure.Configuration, channel *bool) {
	log.Trace("CoreEngine | START")
	var round float64

	lineToPrint := *logCfg.MinLinesToPrint
	for {
		for i := 0; i < len(fileList); i++ { // Iterating the list of file for detecting changes ...
			timestamp := fileutils.GetFileModification(fileList[i].LogFileInfoStruct.Path)                                                       // Get the the last modification of the file
			if (fileList[i].LogFileInfoStruct.Timestamp != timestamp && timestamp != -1 || lineToPrint != *logCfg.MinLinesToPrint) && *channel { // If the lines have changed, update the data
				*channel = false // Close the channel for avoid change of datastructure.Configuration while reading the datastructure.Configuration
				// data := fileutils.Tail(fileList[i].LogFileInfoStruct.Path, -1, 2, *logCfg.MinLinesToPrint)
				// fileList[i].Data = gozstd.Compress(nil, []byte(data))
				fileList[i].Data = utils.ReadFile(fileList[i].LogFileInfoStruct.Path, *logCfg.MinLinesToPrint)
				fileList[i].LogFileInfoStruct.Timestamp = timestamp
				*channel = true // Reopen the channel for be able to change the datastructure.Configuration
				log.Debug("CoreEngine | Round ", round, " | File [", fileList[i].LogFileInfoStruct.Path, "] has changed!!"+
					"[", fileList[i].LogFileInfoStruct.Path, "] Last modification ->", fileList[i].LogFileInfoStruct.Timestamp, " | timestamp -> ", timestamp)
				round++ // Number of time that files have changed
			}
		}
		lineToPrint = *logCfg.MinLinesToPrint
		log.Trace("CoreEngine | Sleeping [", *logCfg.Sleep, "] ZzZzZzZ ....")
		time.Sleep(time.Duration(*logCfg.Sleep) * time.Second)
	}
}

func check(err error) {
	if err != nil {
		log.Warning("ERR: {" + err.Error() + "}")
	}
}

/* ------------- API METHOD ------------- */

// HandleRequests is the hook the real function/wrapper for expose the API. It's main scope it's to map the url to the function that have to do the work.
// It take in input the pointer to the list of file to server; The pointer to the datastructure.Configuration in order to change the parameter at runtime;the channel used for thread safety
func HandleRequests(fileList []datastructure.LogFileStruct, logCfg *datastructure.Configuration, channel *bool) {
	log.Trace("HandleRequests | START")
	m := func(ctx *fasthttp.RequestCtx) { // Hook to the API methods "magilogically"
		ctx.Response.Header.Set("GoLog-Viewer", "v0.0.1$/beta") // Set an header just for track the version of the software
		log.Info("REQUEST --> ", ctx, " | Headers: ", ctx.Request.Header.String())
		tmpChar := "============================================================"
		switch string(ctx.Path()) {
		case "/benchmark":
			fastBenchmarkHTTP(ctx) // Benchmark API
		case "/":
			FastHomePage(ctx, fileList, *logCfg.Hostname, strconv.Itoa(*logCfg.Port)) // Simply print some link
			log.Info(tmpChar)
		case "/listAllFile":
			ListAllFilesHTTP(ctx, fileList) // List all file managed by the application
			log.Info(tmpChar)
		case "/getFile":
			FastGetFileHTTP(ctx, fileList) // Expose the log file
			log.Info(tmpChar)
		case "/filterFromFile":
			FastFilterFileHTTP(ctx, fileList, logCfg) // Filter text from log file
			log.Info(tmpChar)
		case "/changeLine":
			FastChangeLineHTTP(ctx, logCfg, channel) // Change the number of line printed
			log.Info(tmpChar)
		case "/getLinePrinted":
			FastGetLinePrintedHTTP(ctx, logCfg) // Simply print the active datastructure.Configuration parameter
			log.Info(tmpChar)
		default:
			_, err := ctx.WriteString("The url " + string(ctx.URI().RequestURI()) + " does not exist :(\n")
			check(err)
			FastHomePage(ctx, fileList, *logCfg.Hostname, strconv.Itoa(*logCfg.Port)) // Simply print some link
			log.Info(tmpChar)
		}
	}

	// The gzipHandler will serve a compress request only if the client request it with headers (Content-Type: gzip, deflate)
	gzipHandler := fasthttp.CompressHandlerLevel(m, fasthttp.CompressBestCompression)            // Compress data before sending (if requested by the client)
	err := fasthttp.ListenAndServe(*logCfg.Hostname+":"+strconv.Itoa(*logCfg.Port), gzipHandler) // Try to start the server with input "host:port" received in input
	if err != nil {                                                                              // No luck, connection not succesfully. Probably port used ...
		log.Warn("Port ", *logCfg.Port, " seems used :/")
		for i := 0; i < 10; i++ {
			port := strconv.Itoa(utils.Random(8081, 8090)) // Generate a new port to use
			log.Info("Round ", strconv.Itoa(i), "]No luck! Connecting to anotother random port [@", port, "] ...")
			*logCfg.Port, err = strconv.Atoi(port) // Updating the datastructure.Configuration with the new port used
			if err != nil {
				log.Error("HandleRequests | Unable to parse int [", logCfg.Port, "] | Err: ", err)
				return
			}
			err := fasthttp.ListenAndServe(*logCfg.Hostname+":"+port, gzipHandler) // Trying with the random port generate few step above
			if err == nil {                                                        // Connection estabileshed!
				log.Warning("HandleRequests | Connection estabilished @[", *logCfg.Hostname, ":", *logCfg.Port) // Not reached
				break
			}
		}
	}
	log.Trace("HandleRequests | STOP")
}

// FastHomePage is the methods for serve the home page. It print the list of file that you can query with the complete link in order to copy and paste easily
func FastHomePage(ctx *fasthttp.RequestCtx, fileList []datastructure.LogFileStruct, hostname, port string) {
	log.Trace("FastHomePage | START")
	ctx.Response.Header.SetContentType("text/plain; charset=utf-8")
	_, err := ctx.WriteString("Welcome to the GoLog Viewer!\n" + "API List!\n" +
		"http://" + hostname + ":" + port + "/listAllFile -> Return all file managed in a json format\n" +
		"http://" + hostname + ":" + port + "/getFile?file=file_name&json=on -> Return the file log lines (optional: json)\n" +
		"http://" + hostname + ":" + port + "/filterFromFile?file=file_name&filter=toFilter&reverse=on&json=on -> Filter text from the given file (optional: reverse, json)\n" +
		"http://" + hostname + ":" + port + "/changeLine?line=100&json=on -> Change the number of line printed to 100 (optional: json) \n" +
		"http://" + hostname + ":" + port + "/getLinePrinted?json=on -> Return the number of line printed for every log (optional: json)\n")
	check(err)
	var buffer bytes.Buffer // create a buffer for the string content

	for i := 0; i < len(fileList); i++ {
		buffer.WriteString("http://" + hostname + ":" + port + "/getFile?file=" + fileList[i].LogFileInfoStruct.Path + "\n") // append data to the buffer
	}
	_, err = ctx.Write(buffer.Bytes()) // Print the list of the file in the browser
	check(err)
	log.Trace("FastHomePage | STOP")
}

// ListAllFilesHTTP Return a json list of every file saved in the structure
func ListAllFilesHTTP(ctx *fasthttp.RequestCtx, fileList []datastructure.LogFileStruct) {
	log.Trace("ListAllFilesHTTP | START")
	ctx.Response.Header.SetContentType("application/json; charset=utf-8")
	tmpStruct := make([]datastructure.LogFileInfoStruct, len(fileList))
	for i := 0; i < len(fileList); i++ {
		tmpStruct[i] = fileList[i].LogFileInfoStruct
	}
	err := json.NewEncoder(ctx).Encode(datastructure.Status{Status: true, Description: "", ErrorCode: "", Data: tmpStruct})
	check(err)
	log.Debug("ListAllFilesHTTP | Params -> ", string(ctx.QueryArgs().QueryString()), "\nlistAllFilesHTTP | STOP")
}

// FastGetFileHTTP is in charged to find the file related to the INPUT parameter and expose the file over HTTP
func FastGetFileHTTP(ctx *fasthttp.RequestCtx, fileList []datastructure.LogFileStruct) {
	log.Trace("FastGetFileHTTP | START")
	file := string(ctx.FormValue("file")) // Extracting the "file" INPUT parameter
	if strings.Compare(file, "") == 0 {
		ctx.Response.Header.SetContentType("application/json; charset=utf-8")
		err := json.NewEncoder(ctx).Encode(datastructure.Status{Status: false, Description: "Example: /getFile?file=file_name", ErrorCode: "Parameter not found: file", Data: nil})
		check(err)
		log.Error("FastGetFileHTTP without file paramater!")
		log.Trace("FastGetFileHTTP | STOP")
		return
	}
	for i := 0; i < len(fileList); i++ { // Try to find the file ...
		if strings.Compare(fileList[i].LogFileInfoStruct.Path, file) == 0 { // File found !
			dataUncompressed, err := gozstd.Decompress(nil, fileList[i].Data) // Decompress the data
			if err != nil {
				ctx.Response.Header.SetContentType("application/json; charset=utf-8")
				err := json.NewEncoder(ctx).Encode(datastructure.Status{Status: false, Description: "Unable to decompress file " + file, ErrorCode: "UNABLE_DECOMPRESS", Data: nil})
				check(err)
				log.Error("FastGetFileHTTP unable to decompress file " + file)
				log.Trace("FastGetFileHTTP | STOP")
				return

			}
			strJSON := strings.ToLower(string(ctx.FormValue("json")))
			if strings.Compare(strJSON, "on") == 0 || strings.Compare(strJSON, "true") == 0 { // Checking if the json is on
				log.Debug("FastGetFileHTTP | Setting json headers and writing the response")
				ctx.Response.Header.SetContentType("application/json; charset=utf-8")
				err := json.NewEncoder(ctx).Encode(datastructure.Status{Status: true, Description: "", ErrorCode: "", Data: map[string]string{"Name": fileList[i].FileName, "Data": string(dataUncompressed), "Timestamp": strconv.FormatInt(fileList[i].LogFileInfoStruct.Timestamp, 10)}})
				check(err)
			} else {
				log.Debug("FastGetFileHTTP | Setting plain headers and writing the response")
				ctx.Response.Header.SetContentType("text/plain; charset=utf-8")
				_, err := ctx.Write(dataUncompressed)
				check(err)
			}
			log.Info("FastGetFileHTTP | File Found -> ", file, " | Params -> ", ctx)
			log.Trace("FastGetFileHTTP | STOP")
			return
		}
	}
	ctx.Response.Header.SetContentType("application/json; charset=utf-8")
	err := json.NewEncoder(ctx).Encode(datastructure.Status{Status: false, Description: file, ErrorCode: "File not found", Data: nil})
	check(err)
	log.Warn("FastGetFileHTTP | File NOT Found -> ", file, " | Params -> ", string(ctx.QueryArgs().QueryString()))
	log.Trace("FastGetFileHTTP | STOP")
}

// FastFilterFileHTTP is in charge to return the lines of log that contains some text in input and expose the result over HTTP.
// The purpouse of this method is to extract only the lines that contains "filter" from "file" (input parameter)
func FastFilterFileHTTP(ctx *fasthttp.RequestCtx, fileList []datastructure.LogFileStruct, logCfg *datastructure.Configuration) {
	log.Trace("FastFilterFileHTTP | START")
	file := string(ctx.FormValue("file"))                                   // Extracting the "file" INPUT parameter
	filter := string(ctx.FormValue("filter"))                               // Extracting the "filter" INPUT parameter
	var reverse bool                                                        // Used for save the search criteria status
	if strings.Compare(file, "") == 0 || strings.Compare(filter, "") == 0 { // The input parameters are not populated.
		ctx.Response.Header.SetContentType("application/json; charset=utf-8")
		err := json.NewEncoder(ctx).Encode(datastructure.Status{Status: false, Description: "/filterFromFile?file=file_name&filter=to_filter", ErrorCode: "Parameter not found: file,filter", Data: nil})
		check(err)
		log.Warn("FastFilterFileHTTP | Empty file parameters | Params -> ", string(ctx.QueryArgs().QueryString()))
		log.Trace("FastFilterFileHTTP | STOP !")
		return
	}
	tmpReverse := strings.ToLower(string(ctx.FormValue("reverse"))) // Check if the user want to search "everything except that"
	if strings.Compare(tmpReverse, "on") == 0 || strings.Compare(tmpReverse, "true") == 0 {
		reverse = true
	} else {
		reverse = false
	}

	strJSON := strings.ToLower(string(ctx.FormValue("json"))) // Extracting the "json" INPUT parameter
	filteredData := FastFilterFilteHTTPEngine(fileList, *logCfg.MaxLinesToSearch, &file, &filter, reverse)
	if strings.Compare(strJSON, "on") == 0 || strings.Compare(strJSON, "true") == 0 { // Checking if the json is on
		log.Trace("FastFilterFileHTTP | Setting json headers and writing the response")
		ctx.Response.Header.SetContentType("application/json; charset=utf-8")
		err := json.NewEncoder(ctx).Encode(datastructure.Status{Status: true, Description: "", ErrorCode: "", Data: filteredData})
		check(err)
	} else {
		log.Trace("FastFilterFileHTTP | Setting plain headers and writing the response")
		ctx.Response.Header.SetContentType("text/plain; charset=utf-8")
		_, err := ctx.WriteString(filteredData)
		check(err)
	}
	log.Info("FastFilterFileHTTP | Hit with -> ", filter, " | ", file, " | Params -> ", string(ctx.QueryArgs().QueryString()))
	log.Trace("FastFilterFileHTTP | STOP")
}

// FastFilterFilteHTTPEngine is a wrapper for the core logic method
func FastFilterFilteHTTPEngine(fileList []datastructure.LogFileStruct, maxLinesToSearch int, file *string, filter *string, reverse bool) string {
	log.Trace("FastFilterFilteHTTPEngine | START")
	for i := 0; i < len(fileList); i++ {
		if strings.Compare(fileList[i].LogFileInfoStruct.Path, *file) == 0 { // Try to find the file
			log.Debug("FastFilterFilteHTTPEngine | File found! | Filtering " + *filter + " from " + fileList[i].LogFileInfoStruct.Path)
			filteredData := utils.FilterFromFile(fileList[i].LogFileInfoStruct.Path, maxLinesToSearch, *filter, reverse)
			log.Debug("FastFilterFilteHTTPEngine | Filtered data -> " + filteredData)
			return filteredData
		}
	}
	log.Warn("FastFilterFilteHTTPEngine | File not found :/ | STOP")
	return ""
}

// FastChangeLineHTTP API for change the line printed @runtime
func FastChangeLineHTTP(ctx *fasthttp.RequestCtx, logCfg *datastructure.Configuration, channel *bool) {
	log.Trace("FastChangeLineHTTP | START")
	ctx.Response.Header.SetContentType("application/json; charset=utf-8")
	line := string(ctx.FormValue("line"))
	if strings.Compare(line, "") == 0 {
		log.Error("FastChangeLineHTTP | Request failed! Missing parameter! | Request -> ", ctx)
		err := json.NewEncoder(ctx).Encode(datastructure.Status{Status: false, Description: "/changeLine?line=200", ErrorCode: "Parameter not found; line", Data: logCfg})
		check(err)
		log.Trace("FastChangeLineHTTP | STOP")
		return
	}
	n, err := strconv.Atoi(line) // Convert INPUT string to int
	if err != nil {
		log.Error("FastChangeLineHTTP | Request failed! Int conversion failed :(!!", err)
		err := json.NewEncoder(ctx).Encode(datastructure.Status{Status: false, Description: err.Error(), ErrorCode: "Int conversion failed", Data: logCfg})
		check(err)
		log.Trace("FastChangeLineHTTP | STOP")
		return
	}

	if !*channel {
		log.Error("FastChangeLineHTTP | Request failed! CoreEngine process is running !!")
		err := json.NewEncoder(ctx).Encode(datastructure.Status{Status: false, Description: "Core engine running", ErrorCode: "Channel busy", Data: logCfg})
		check(err)
		log.Trace("FastChangeLineHTTP | STOP")
		return
	}
	*channel = false            // Closing the channel for change the datastructure.Configuration
	*logCfg.MinLinesToPrint = n // Apply the datastructure.Configuration
	*channel = true             // datastructure.Configuration applied! Reopening the channel
	err = json.NewEncoder(ctx).Encode(datastructure.Status{Status: true, Description: "datastructure.Configuration changed to " + line, ErrorCode: "", Data: logCfg})
	check(err)
	log.Warn("FastChangeLineHTTP | Request succed -> Changing to " + strconv.Itoa(n))
	log.Trace("FastChangeLineHTTP | STOP")
}

// FastGetLinePrintedHTTP return the number of line printed
func FastGetLinePrintedHTTP(ctx *fasthttp.RequestCtx, logCfg *datastructure.Configuration) {
	log.Trace("FastGetLinePrintedHTTP | START")
	ctx.Response.Header.SetContentType("application/json; charset=utf-8")
	err := json.NewEncoder(ctx).Encode(datastructure.Status{Status: true, Description: "", ErrorCode: "", Data: logCfg})
	check(err)
	log.Trace("FastGetLinePrintedHTTP | STOP")
}

// FastGetLinePrintedHTTP return the number of line printed
func fastBenchmarkHTTP(ctx *fasthttp.RequestCtx) {
	_, err := ctx.Write([]byte("Retry !"))
	check(err)
}

/* ------------- datastructure.Configuration METHOD ------------- */

// InitConfigurationData is in charge to init the various datastructure.Configuration data.
// It runs only once for load the data and instantiate datastructure.Configuration options.
func InitConfigurationData() datastructure.Configuration {
	log.Trace("Initdatastructure.ConfigurationData | START")
	var (
		logPath  string // User INPUT folder
		numLines int    // Number of line to print
		maxLines int    // Max lines used for search the content
		port     int    // Port to bind the service (random from 8080/8090 if used)
		host     string // Host to bind the service (127.0.0.1 as default)
		sleep    int    // Seconds for wait until check the log another time
		gcSleep  int    // Number of minutes to sleep beetween every forced GC cycle
	)

	logPath, numLines, maxLines, port, host, sleep, gcSleep = VerifyCommandLineInput() // Function for validate command line INPUT
	if strings.Compare(logPath[len(logPath)-1:], "/") != 0 {                           // Be sure that the last character is an '/'
		logPath += "/" // Append the character needed by the directory if not present
	}
	// Init a new datastructure.Configuration
	log.Trace("Initdatastructure.ConfigurationData | STOP")
	return datastructure.Configuration{Path: &logPath, MinLinesToPrint: &numLines, MaxLinesToSearch: &maxLines, Port: &port, Hostname: &host, Sleep: &sleep, GCSleep: &gcSleep}
}

// VerifyCommandLineInput verify about the INPUT parameter passed as arg[]
func VerifyCommandLineInput() (string, int, int, int, string, int, int) {
	log.Trace("VerifyCommandLineInput | START")
	path := flag.String("path", "", "Log folder (MANDATORY PARAMETER)")
	linesFlag := flag.Int("lines", 200, "Lines to filter")
	maxLines := flag.Int("maxlines", 100000, "Max lines used for filter")
	port := flag.Int("port", 8080, "Port to bind the service")
	host := flag.String("host", "localhost", "Host to bind the service")
	sleep := flag.Int("sleep", 15, "Seconds for wait until another iteration")
	gcSleep := flag.Int("gcSleep", 5, "Number of minutes to sleep beetween every forced GC cycle")
	flag.Parse()
	if strings.Compare(*path, "") == 0 { // Verify that "path" (INPUT parameter) is populated
		flag.PrintDefaults() // Exit status 2, bye bye Sir
		log.Fatal("Start without -path parameter :/")

	}
	log.Trace("VerifyCommandLineInput | Starting command line input validation ..")
	if utils.IsDir(*path) { // Be sure that the INPUT directory exist and parse the lines
		// #TODO: these check are unusefull, due to the default value assigned. Are just a template for a future "input validation methods"
		if *linesFlag == 0 { // If no lines provided set 1000 as standard output lines
			*linesFlag = 1000
			log.Warn("VerifyCommandLineInput | Use -lines 2000 if you want to choose to print 2000 lines")
		}
		if *maxLines == 0 { // If no lines provided select 1000 as default search lines for text
			*maxLines = 1000000
			log.Warn("VerifyCommandLineInput | Use -maxlines 1000000 to choose search the text among 1000000 lines ")
		}
		if *port == 0 { // If no port selected, generate select a random one from 8080 to 8090
			*port = utils.Random(8081, 8090)
			log.Error("VerifyCommandLineInput | Use -port 8081 to bind the service on the port 8081 | Binded @", *port)
		}
		if stringutils.IsBlank(*host) {
			*host = "localhost" //if no host provided set localhost
			log.Error("VerifyCommandLineInput | Use -host localhost for bind the service to 127.0.0.1 | Binded @", *host)
		}
		log.Info("INPUT folder: ", *path, " | Lines to print: ", strconv.Itoa(*linesFlag), " | Max line to filter: ", strconv.Itoa(*maxLines),
			" | Port: ", *port, " | Host: ", *host, " | Sleep: ", *sleep, " | GCSleep: ", *gcSleep)
		log.Trace("VerifyCommandLineInput | STOP")
		return *path, *linesFlag, *maxLines, *port, *host, *sleep, *gcSleep
	}
	log.Fatal("VerifyCommandLineInput | ERROR: No folder found like ", *path)
	return "", 0, 0, 0, "", 0, 0 // Unuseful, Fatal will call os.Exit(1)
}

// InitLogFileData Init the log file. It runs only once for load the data and instantiate the array of logfile
func InitLogFileData(logCfg *datastructure.Configuration) []datastructure.LogFileStruct {
	log.Debug("InitLogFileData | START")
	var filesList []string                    // Save the list of file name
	var logList []datastructure.LogFileStruct // Structure that have to be returned
	// rawFilesList := utils.ReadFilePath(*logCfg.Path) // Get the list of the file in the directory
	rawFilesList := fileutils.ListFile(*logCfg.Path) // Get the list of the file in the directory
	if len(rawFilesList) == 0 {
		log.Fatal("`Impossible` to access into -> ", *logCfg.Path) // Bye bye
		return nil
	}

	for _, item := range rawFilesList {
		fileType, err := fileutils.GetFileContentType(item)
		if err != nil {
			log.Warning("Error for file [" + item + "] -> Err: " + err.Error())
		} else {
			if strings.HasPrefix(fileType, "text/plain") {
				filesList = append(filesList, item)
			} else {
				log.Warning("File type for file [" + item + "] -> " + fileType)
			}
		}
	}

	log.Info("List of file in logpath -> ", filesList, " | Number of files -> ", len(filesList))
	filesLen := len(filesList)
	logList = make([]datastructure.LogFileStruct, filesLen) // Allocate an array of LogFileStruct
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 64)
	wg.Add(filesLen)
	for i := 0; i < filesLen; i++ { // Populate with the data
		go func(i int) {
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			defer wg.Done()
			tmpname := strings.Split(filesList[i], "/")   // Tokenize the string by the /
			logList[i].FileName = tmpname[len(tmpname)-1] // Extract only the Name of the file (latest element after "/")
			logList[i].LogFileInfoStruct.Path = filesList[i]
			logList[i].Data = utils.ReadFile(filesList[i], *logCfg.MinLinesToPrint)
			logList[i].LogFileInfoStruct.Timestamp = fileutils.GetFileModification(filesList[i])
		}(i)
		//fmt.Printf("\r %d/%d - %s", i, filesLen, logList[i].FileName)
	}
	wg.Wait()
	log.Debug("InitLogFileData | STOP")
	return logList
}
