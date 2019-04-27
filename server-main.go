package main

// Copyright (C) 2017-2019 Philip Schlump.  See ./LICENSE

// Sample: http://127.0.0.1:9018/api/v1/genpdf?in=https://www.google.com&title=bo

// xyzzy - "in" should be URL decoded.
// xyzzy - send output in PDF format back to caller insetead of seting JSON with path.
// xyzzy - should track number of errors and where for status return.

import (
	"context"
	"crypto/tls"
	"database/sql"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"time"

	"github.com/American-Certified-Brands/config-sample/ReadConfig"
	"github.com/American-Certified-Brands/tools/CliResponseWriter"
	"github.com/American-Certified-Brands/tools/GetVar" // pdf "github.com/adrg/go-wkhtmltopdf"
	"github.com/American-Certified-Brands/tools/apache_logger"
	"github.com/American-Certified-Brands/tools/lms"
	"github.com/pschlump/HashStrings"
	"github.com/pschlump/MiscLib"
	"github.com/pschlump/godebug" // "github.com/pschlump/mon-alive/lib"
	template "github.com/pschlump/textTemplate"
	"github.com/pschlump/uuid"
)

// ----------------------------------------------------------------------------------
//
// Notes:
//   Graceful Shutdown: https://stackoverflow.com/questions/39320025/how-to-stop-http-listenandserve
//   Email with HTML and Text: https://stackoverflow.com/questions/9950098/how-to-send-an-email-using-go-with-an-html-formatted-body
//
// Install of wkhtmltopdf on ubuntu
// 		$ apt-get install xvfb libfontconfig wkhtmltopdf
// In Makefile See:
// # From: https://github.com/mileszs/wicked_pdf/issues/723
// NoteInstallPatchedVersion:
// 		wget https://github.com/wkhtmltopdf/wkhtmltopdf/releases/download/0.12.3/wkhtmltox-0.12.3_linux-generic-amd64.tar.xz
//
// ----------------------------------------------------------------------------------

var Cfg = flag.String("cfg", "cfg.json", "config file for this call")
var Cli = flag.String("cli", "", "Run as a CLI command intead of a server")
var HostPort = flag.String("hostport", ":9021", "Host/Port to listen on")
var DbFlag = flag.String("db_flag", "", "Additional Debug Flags")
var TLS_crt = flag.String("tls_crt", "", "TLS Signed Publick Key")
var TLS_key = flag.String("tls_key", "", "TLS Signed Private Key")

type GlobalConfigData struct {
	lms.BaseConfigType

	OutputPath  string `json:"OutputPath" default:"./www/out"`
	OutputURI   string `json:"OutputURI" default:"/out"`
	WkHTMLToPdf string `json:"WkHTMLToPdf" default:"/usr/local/bin/wkhtmltopdf"`
}

var gCfg GlobalConfigData
var GitCommit string
var nPDFConverted = 0
var nPDFConvertedMux *sync.Mutex
var logFilePtr *os.File
var DB *sql.DB
var db_flag map[string]bool
var wg sync.WaitGroup
var httpServer *http.Server
var logger *log.Logger
var shutdownWaitTime = time.Duration(1)
var isTLS bool
var wd string

func init() {
	isTLS = false
	nPDFConvertedMux = &sync.Mutex{}
	db_flag = make(map[string]bool)
	logger = log.New(os.Stdout, "", 0)
	template.SetNoValue("", true)
}

func main() {
	// pdf.Init()
	// defer pdf.Destroy()

	wd = GetWD()

	flag.Parse() // Parse CLI arguments to this, --cfg <name>.json

	fns := flag.Args()
	if *Cli != "" {
		GetVar.SetCliOpts(Cli, fns)
	} else if len(fns) != 0 {
		fmt.Printf("Extra arguments are not supported [%s]\n", fns)
		os.Exit(1)
	}

	if Cfg == nil {
		fmt.Printf("--cfg is a required parameter\n")
		os.Exit(1)
	}

	// ------------------------------------------------------------------------------
	// Read in Configuraiton
	// ------------------------------------------------------------------------------
	err := ReadConfig.ReadFile(*Cfg, &gCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read confguration: %s error %s\n", *Cfg, err)
		os.Exit(1)
	}

	// ------------------------------------------------------------------------------
	// Logging File
	// ------------------------------------------------------------------------------
	logFilePtr = lms.LoggingSetup(&(gCfg.BaseConfigType))

	// ------------------------------------------------------------------------------
	// TLS parameter check / setup
	// ------------------------------------------------------------------------------
	lms.TLSSetup(TLS_crt, TLS_key, &isTLS, &(gCfg.BaseConfigType))

	// ------------------------------------------------------------------------------
	// Debug Flag Processing
	// ------------------------------------------------------------------------------
	lms.DebugFlagProcess(DbFlag, db_flag, &(gCfg.BaseConfigType))
	GetVar.SetDbFlag(db_flag)
	CliResponseWriter.SetDbFlag(db_flag)

	// ------------------------------------------------------------------------------
	// Setup HTTP End Points
	// ------------------------------------------------------------------------------
	mux := http.NewServeMux()
	mux.Handle("/api/v1/status", http.HandlerFunc(HandleStatus))          //
	mux.Handle("/status", http.HandlerFunc(HandleStatus))                 //
	mux.Handle("/api/v1/exit-server", http.HandlerFunc(HandleExitServer)) //
	mux.Handle("/api/v1/genpdf", http.HandlerFunc(HandleGenPDF))          //
	mux.Handle("/", http.FileServer(http.Dir(gCfg.StaticPath)))

	// ------------------------------------------------------------------------------
	// Setup signal capture
	// ------------------------------------------------------------------------------
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	// ------------------------------------------------------------------------------
	// Live Monitor Setup
	// ------------------------------------------------------------------------------
	lms.LiveMonSetup("PDF-Generate-MS", db_flag, &(gCfg.BaseConfigType))

	// ------------------------------------------------------------------------------
	// ------------------------------------------------------------------------------
	if *Cli != "" {
		www := CliResponseWriter.NewCliResonseWriter() // www := http.ResponseWriter
		/*
		   type url.URL struct {
		   	Scheme     string
		   	Opaque     string    // encoded opaque data
		   	User       *Userinfo // username and password information
		   	Host       string    // host or host:port
		   	Path       string    // path (relative paths may omit leading slash)
		   	RawPath    string    // encoded path hint (see EscapedPath method)
		   	ForceQuery bool      // append a query ('?') even if RawQuery is empty
		   	RawQuery   string    // encoded query values, without '?'
		   	Fragment   string    // fragment for references, without '#'
		   }
		   // From: https://golang.org/src/net/url/url.go?s=9736:10252#L353 :363
		*/
		qryParam := GetVar.GenQryFromCli()
		if db_flag["cli"] {
			fmt.Printf("qry_params= ->%s<- at:%s\n", qryParam, godebug.LF())
		}
		u := url.URL{
			User:     nil,
			Host:     "127.0.0.1:80",
			Path:     *Cli,
			RawQuery: qryParam,
		}
		req := &http.Request{ // https://golang.org/src/net/http/request.go:113
			Method:     "GET",
			URL:        &u, // *url.URL
			Proto:      "HTTP/1.0",
			ProtoMajor: 1,
			ProtoMinor: 0,
			Header:     make(http.Header),
			// Body io.ReadCloser // :182 -- not used, GET request - no body.
			// Form url.Values -- Populate with values from CLI
			Host:       "127.0.0.1:80",
			RequestURI: *Cli + "?" + qryParam, // "RequestURI": "/api/v1/status?id=dump-request",
		}
		switch *Cli {
		case "/api/v1/status":
			HandleStatus(www, req)
		case "/api/v1/exit-server":
			fmt.Printf("Exit server\n")
		case "/api/v1/genpdf":
			HandleGenPDF(www, req)
		default:
			fn := "./www/" + *Cli
			s, err := ioutil.ReadFile(fn)
			if err != nil {
				fmt.Printf("Status: 404\n")
			} else {
				fmt.Printf("Status: 200\n")
				fmt.Printf("%s\n", s)
			}
		}
		www.Flush()
		if db_flag["Cli.Where"] {
			www.DumpWhere()
		}
		os.Exit(0)
	}

	// ------------------------------------------------------------------------------
	// Setup / Run the HTTP Server.
	// ------------------------------------------------------------------------------
	if isTLS {
		cfg := &tls.Config{
			MinVersion:               tls.VersionTLS12,
			CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
			PreferServerCipherSuites: true,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			},
		}
		loggingHandler := apache_logger.NewApacheLoggingHandler(mux, logFilePtr, nil, &apache_logger.DBACleanupFunc)
		httpServer = &http.Server{
			Addr:         *HostPort,
			Handler:      loggingHandler,
			TLSConfig:    cfg,
			TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
		}
	} else {
		loggingHandler := apache_logger.NewApacheLoggingHandler(mux, logFilePtr, nil, &apache_logger.DBACleanupFunc)
		httpServer = &http.Server{
			Addr:    *HostPort,
			Handler: loggingHandler,
		}
	}

	go func() {
		wg.Add(1)
		defer wg.Done()
		if isTLS {
			fmt.Fprintf(os.Stderr, "%sListening on https://%s%s\n", MiscLib.ColorGreen, *HostPort, MiscLib.ColorReset)
			if err := httpServer.ListenAndServeTLS(*TLS_crt, *TLS_key); err != nil {
				logger.Fatal(err)
			}
		} else {
			fmt.Fprintf(os.Stderr, "%sListening on http://%s%s\n", MiscLib.ColorGreen, *HostPort, MiscLib.ColorReset)
			if err := httpServer.ListenAndServe(); err != nil {
				logger.Fatal(err)
			}
		}
	}()

	// ------------------------------------------------------------------------------
	// Catch signals from [Contro-C]
	// ------------------------------------------------------------------------------
	select {
	case <-stop:
		fmt.Fprintf(os.Stderr, "\nShutting down the server... Received OS Signal...\n")
		ctx, cancel := context.WithTimeout(context.Background(), shutdownWaitTime*time.Second)
		defer cancel()
		err := httpServer.Shutdown(ctx)
		if err != nil {
			fmt.Printf("Error on shutdown: [%s]\n", err)
		}
	}

	// ------------------------------------------------------------------------------
	// Wait for HTTP server to exit.
	// ------------------------------------------------------------------------------
	wg.Wait()
}

func IncPdf() {
	nPDFConvertedMux.Lock()
	nPDFConverted = 0
	nPDFConvertedMux.Unlock()
}

func HandleGenPDF(www http.ResponseWriter, req *http.Request) {
	var err error
	if isTLS {
		www.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
	}
	if db_flag["file-names"] {
		fmt.Printf(" At Top: %s\n", godebug.LF())
	}

	if !CheckAuthToken(www, req) {
		www.WriteHeader(http.StatusUnauthorized) // 401
		fmt.Fprintf(www, "Error: not authorized.\n")
		return
	}

	found_in, in := GetVar.GetVar("in", www, req)
	found_title, title := GetVar.GetVar("title", www, req)
	if !found_title {
		title = "Genearted PDF From: " + in
	}

	_ = title

	if !found_in {
		www.WriteHeader(http.StatusBadRequest)
		return
	}

	id0, _ := uuid.NewV4()
	tmpFn := id0.String()

	genTmp := ""
	if gCfg.OutputPath[0:1] == "/" {
		genTmp = fmt.Sprintf("%s/%s.pdf", gCfg.OutputPath, tmpFn)
		// fmt.Fprintf(os.Stderr, "AT: %s\n", godebug.LF())
	} else {
		genTmp = fmt.Sprintf("%s/%s/%s.pdf", wd, gCfg.OutputPath, tmpFn)
		// fmt.Fprintf(os.Stderr, "AT: %s\n", godebug.LF())
	}

	if db_flag["file-names"] {
		fmt.Printf(" At Top: %s genTmp=[%s]\n", godebug.LF(), genTmp)
	}

	//	if db_flag["use-wkhtmltopdf-library"] { // set to true if the Go WkHTMLToPDF library works.
	//		fmt.Fprintf(os.Stderr, "AT: %s\n", godebug.LF())
	//		// err = GenPDF("yep yep yep", in, genTmp)
	//		err = GenPDF(title, in, genTmp)
	//		fmt.Fprintf(os.Stderr, "AT: %s, err %s\n", godebug.LF(), err)
	//	} else {
	// fmt.Fprintf(os.Stderr, "AT: %s\n", godebug.LF())
	err = RunGenPDF(in, genTmp)
	// fmt.Fprintf(os.Stderr, "AT: %s, err %s\n", godebug.LF(), err)
	//	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s AT:%s\n", err, godebug.LF())
		www.WriteHeader(http.StatusInternalServerError)
	}

	// fmt.Fprintf(os.Stderr, "AT: %s\n", godebug.LF())
	data, err := ioutil.ReadFile(genTmp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s AT:%s\n", err, godebug.LF())
		www.WriteHeader(http.StatusInternalServerError)
		return
	}

	var newFn, newURI string
	// fmt.Fprintf(os.Stderr, "AT: %s\n", godebug.LF())
	hash := HashStrings.HashByte(data)
	if gCfg.OutputPath[0:1] == "/" {
		newFn = fmt.Sprintf("%s/%x.pdf", gCfg.OutputPath, hash)
	} else {
		newFn = fmt.Sprintf("%s/%s/%x.pdf", wd, gCfg.OutputPath, hash)
	}
	newURI = fmt.Sprintf("%s/%x.pdf", gCfg.OutputURI, hash)
	if db_flag["file-names"] {
		fmt.Printf("\n%sAt Top: %s%s\n\tgenTmp=[%s]\n\tnewFn=[%s]\n\tnewURI=[%s]\n\n", MiscLib.ColorYellow, godebug.LF(), MiscLib.ColorReset, genTmp, newFn, newURI)
	}
	// fmt.Fprintf(os.Stderr, "AT: %s\n", godebug.LF())
	err = os.Rename(genTmp, newFn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s AT:%s\n\tFrom[%s]\n\tTo   [%s]\n\n", err, godebug.LF(), genTmp, newFn)
		www.WriteHeader(http.StatusInternalServerError)
		return
	}

	// fmt.Fprintf(os.Stderr, "AT: %s\n", godebug.LF())
	www.Header().Set("Content-Type", "application/json; charset=utf-8")
	www.WriteHeader(http.StatusOK) // 200
	fmt.Fprintf(www, `{"status":"success","URI":%q}`+"\n", newURI)
	return
}

// HandleStatus - server to respond with a working message if up.
func HandleStatus(www http.ResponseWriter, req *http.Request) {
	found, resetToZero := GetVar.GetVar("resetToZero", www, req)
	if isTLS {
		www.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
	}
	if found && resetToZero == "yes" {
		nPDFConvertedMux.Lock()
		nPDFConverted = 0
		nPDFConvertedMux.Unlock()
	}
	www.Header().Set("Content-Type", "application/json; charset=utf-8")
	www.WriteHeader(http.StatusOK) // 200
	fmt.Fprintf(www, `{"status":"success", "version":%q, "nPDFConverted":%d}`+"\n", GitCommit, nPDFConverted)
	return
}

// HandleExitServer - graceful server shutdown.
func HandleExitServer(www http.ResponseWriter, req *http.Request) {

	if !lms.IsAuthKeyValid(www, req, &(gCfg.BaseConfigType)) {
		return
	}
	if isTLS {
		www.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
	}
	www.Header().Set("Content-Type", "application/json; charset=utf-8")

	www.WriteHeader(http.StatusOK) // 200
	fmt.Fprintf(www, `{"status":"success", "version":%q, "nPDFConverted":%d}`+"\n", GitCommit, nPDFConverted)

	go func() {
		// Implement graceful exit with auth_key
		fmt.Fprintf(os.Stderr, "\nShutting down the server... Received /exit-server?auth_key=...\n")
		ctx, cancel := context.WithTimeout(context.Background(), shutdownWaitTime*time.Second)
		defer cancel()
		err := httpServer.Shutdown(ctx)
		if err != nil {
			fmt.Printf("Error on shutdown: [%s]\n", err)
		}
	}()
}

func GetWD() string {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	return dir
}

func RunGenPDF(in, out string) (err error) {
	cmd := exec.Command(gCfg.WkHTMLToPdf, in, out)
	if db_flag["print-command-success"] {
		fmt.Printf("Running command and waiting for it to finish...")
	}
	IncPdf()
	err = cmd.Run()
	if err != nil {
		fmt.Printf("Command finished with error: %v: %s %s %s\n", err, gCfg.WkHTMLToPdf, in, out)
	} else {
		if db_flag["print-command-success"] {
			fmt.Printf("Command finished with success: %s %s %s\n", gCfg.WkHTMLToPdf, in, out)
		}
	}
	return
}
