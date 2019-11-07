package datastructure

/* ------------- DATA STRUCTURE ------------- */

// LogFileStruct Base structure for manage log file information
type LogFileStruct struct {
	FileName          string            `json:"Filename"`          // Name of the log file
	Data              []byte            `json:"Data"`              // Compress data of log files
	LogFileInfoStruct LogFileInfoStruct `json:"LogFileInfoStruct"` // Path and timestamp of the logfile
}

// LogFileInfoStruct Base structure for save the metadata inoìformation of the log file
type LogFileInfoStruct struct {
	Timestamp int64  `json:"Timestamp"` // Last modification time of the log file (user for check change)
	Path      string `json:"Path"`      // Path of the log file (symbolic link welcome)
}

// Status Structure used for populate the json response for the RESTfull HTTP API
type Status struct {
	Status      bool        `json:"Status"`      // Status of response [true,false] OK, KO
	ErrorCode   string      `json:"ErrorCode"`   // Code linked to the error (KO)
	Description string      `json:"Description"` // Description linked to the error (KO)
	Data        interface{} `json:"Data"`        // Generic data to return in the response
}

// Configuration Structure for manage the configuration of the tool
type Configuration struct {
	Path             *string `json:"Path"`             // Path of the log folder that have to be scan recursively during the init phase of the configuration
	MinLinesToPrint  *int    `json:"MinLinesToPrint"`  // Minium number of lines to save in memory
	MaxLinesToSearch *int    `json:"MaxLinesToSearch"` // Max line to take care when search (filter) content
	Port             *int    `json:"Port"`             // Port to bind the service
	Hostname         *string `json:"Hostname"`         // Hostname to bind the service
	Sleep            *int    `json:"Sleep"`            // Number of seconds to sleep every time that the "core engine" have scan the filess
	GCSleep          *int    `json:"GCSleep"`          // Number of minutes to sleep among every time that the manual garbage collector is called
}
