package main

//	GenPdf("https://en.wikipedia.org/wiki/Secure_Remote_Password_protocol", ",a.pdf")
//	GenPdf("https://www.google.com", ",b.pdf")

import (
	"context"
	"crypto/tls"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/American-Certified-Brands/config-sample/ReadConfig"
	pdf "github.com/adrg/go-wkhtmltopdf"
	"github.com/pschlump/MiscLib"
	"github.com/pschlump/filelib"
	"github.com/pschlump/godebug"
	MonAliveLib "github.com/pschlump/mon-alive/lib" // "github.com/pschlump/mon-alive/lib"
	"github.com/pschlump/radix.v2/redis"
	template "github.com/pschlump/textTemplate"
)

// ----------------------------------------------------------------------------------
// Notes:
//   Graceful Shutdown: https://stackoverflow.com/questions/39320025/how-to-stop-http-listenandserve
//   Email with HTML and Text: https://stackoverflow.com/questions/9950098/how-to-send-an-email-using-go-with-an-html-formatted-body
// ----------------------------------------------------------------------------------

var Cfg = flag.String("cfg", "cfg.json", "config file for this call")
var HostPort = flag.String("hostport", ":9022", "Host/Port to listen on")
var DbFlag = flag.String("db_flag", "", "Additional Debug Flags")
var TLS_crt = flag.String("tls_crt", "", "TLS Signed Publick Key")
var TLS_key = flag.String("tls_key", "", "TLS Signed Private Key")

type GlobalConfigData struct {
	// Add in Redis stuff
	RedisConnectHost string `json:"redis_host" default:"$ENV$REDIS_HOST"`
	RedisConnectAuth string `json:"redis_auth" default:"$ENV$REDIS_AUTH"`
	RedisConnectPort string `json:"redis_port" default:"6379"`

	LogFileName string `json:"log_file_name"`

	OutputPath string `json:"./www/out"`
	// debug flags:
	DebugFlag string `json:"db_flag"`

	AuthKey string `json:"auth_key" default:""` // Auth key by default is turned off.

	// Default file for TLS setup (Should include path), both must be specified.
	// These can be over ridden on the command line.
	TLS_crt string `json:"tls_crt" default:""`
	TLS_key string `json:"tls_key" default:""`
}

var gCfg GlobalConfigData
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

func init() {
	isTLS = false
	nPDFConvertedMux = &sync.Mutex{}
	db_flag = make(map[string]bool)
	logger = log.New(os.Stdout, "", 0)
	template.SetNoValue("", true)
}

func main() {
	pdf.Init()
	defer pdf.Destroy()

	flag.Parse() // Parse CLI arguments to this, --cfg <name>.json

	fns := flag.Args()
	if len(fns) != 0 {
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
	if gCfg.LogFileName != "" {
		bn := path.Dir(gCfg.LogFileName)
		os.MkdirAll(bn, 0755)
		fp, err := filelib.Fopen(gCfg.LogFileName, "a")
		if err != nil {
			log.Fatalf("log file confiured, but unable to open, file[%s] error[%s]\n", gCfg.LogFileName, err)
		}
		LogFile(fp)
	}

	// ------------------------------------------------------------------------------
	// TLS parameter check / setup
	// ------------------------------------------------------------------------------
	if *TLS_crt == "" && gCfg.TLS_crt != "" {
		TLS_crt = &gCfg.TLS_crt
	}
	if *TLS_key == "" && gCfg.TLS_key != "" {
		TLS_key = &gCfg.TLS_key
	}

	if *TLS_crt != "" && *TLS_key == "" {
		log.Fatalf("Must supply both .crt and .key for TLS to be turned on - fatal error.")
	} else if *TLS_crt == "" && *TLS_key != "" {
		log.Fatalf("Must supply both .crt and .key for TLS to be turned on - fatal error.")
	} else if *TLS_crt != "" && *TLS_key != "" {
		if !filelib.Exists(*TLS_crt) {
			log.Fatalf("Missing file ->%s<-\n", *TLS_crt)
		}
		if !filelib.Exists(*TLS_key) {
			log.Fatalf("Missing file ->%s<-\n", *TLS_key)
		}
		isTLS = true
	}

	// ------------------------------------------------------------------------------
	// Debug Flag Processing
	// ------------------------------------------------------------------------------
	if gCfg.DebugFlag != "" {
		ss := strings.Split(gCfg.DebugFlag, ",")
		// fmt.Printf("gCfg.DebugFlag ->%s<-\n", gCfg.DebugFlag)
		for _, sx := range ss {
			// fmt.Printf("Setting ->%s<-\n", sx)
			db_flag[sx] = true
		}
	}
	if *DbFlag != "" {
		ss := strings.Split(*DbFlag, ",")
		// fmt.Printf("gCfg.DebugFlag ->%s<-\n", gCfg.DebugFlag)
		for _, sx := range ss {
			// fmt.Printf("Setting ->%s<-\n", sx)
			db_flag[sx] = true
		}
	}
	if db_flag["dump-db-flag"] {
		fmt.Fprintf(os.Stderr, "%sDB Flags Enabled Are:%s\n", MiscLib.ColorGreen, MiscLib.ColorReset)
		for x := range db_flag {
			fmt.Fprintf(os.Stderr, "%s\t%s%s\n", MiscLib.ColorGreen, x, MiscLib.ColorReset)
		}
	}

	// ------------------------------------------------------------------------------
	// Setup HTTP End Points
	// ------------------------------------------------------------------------------
	mux := http.NewServeMux()
	mux.Handle("/api/v1/status", http.HandlerFunc(HandleStatus))          //
	mux.Handle("/api/v1/exit-server", http.HandlerFunc(HandleExitServer)) //
	mux.Handle("/api/v1/genpdf", http.HandlerFunc(HandleGenPDF))          //
	mux.Handle("/", http.FileServer(http.Dir("www")))

	// ------------------------------------------------------------------------------
	// Setup signal capture
	// ------------------------------------------------------------------------------
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	// ------------------------------------------------------------------------------
	// Live Monitor Setup
	// ------------------------------------------------------------------------------
	monClient, err7 := RedisClient()
	fmt.Printf("err7=%v AT: %s\n", err7, godebug.LF())
	mon := MonAliveLib.NewMonIt(func() *redis.Client { return monClient }, func(conn *redis.Client) {})
	mon.SendPeriodicIAmAlive("PDF-Generation-MS")

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
		httpServer = &http.Server{
			Addr:         *HostPort,
			Handler:      mux,
			TLSConfig:    cfg,
			TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
		}
	} else {
		httpServer = &http.Server{
			Addr:    *HostPort,
			Handler: mux,
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

// xyzzy - send output in PDF format back to caller insetead of seting JSON with path.
// mux.Handle("/api/v1/genpdf", http.HandlerFunc(HandleGenPDF))     //
func HandleGenPDF(www http.ResponseWriter, req *http.Request) {
	if isTLS {
		www.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
	}

	if !CheckAuthToken(www, req) {
		www.WriteHeader(http.StatusUnauthorized) // 401
		fmt.Fprintf(www, "Error: not authorized.\n")
		return
	}

	found_in, in := GetVar("in", www, req)
	found_title, title := GetVar("title", www, req)
	if !found_title {
		title = "Genearted PDF From: " + in
	}
	found_out, out := GetVar("out", www, req)

	if !found_in {
		www.WriteHeader(http.StatusBadRequest)
		return
	}
	if !found_out {
		www.WriteHeader(http.StatusBadRequest)
		return
	}

	out = fmt.Sprintf("%s/%s", gCfg.OutputPath, out) // should generate a UUID based or SHA256 based filename. xyzzy
	// gen unique file name, temporary, "./www/out/<<tmp>>.pdf"
	// gen PDF to file - add hard part to front of it
	// calc SHA256 for it
	// renamte unique to SHA256.pdf - using hard path
	// return that. return URL based short path.  /out/<<SHA256>>.pdf

	err := GenPDF(title, in, out)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		www.WriteHeader(http.StatusInternalServerError) // is this the correct error to return at this point?
	}

	www.Header().Set("Content-Type", "application/json; charset=utf-8")
	www.WriteHeader(http.StatusOK) // 200
	fmt.Fprintf(www, `{"status":"success","URI":%q}`+"\n", out)
	return
}

// HandleStatus - server to respond with a working message if up.
func HandleStatus(www http.ResponseWriter, req *http.Request) {
	found, resetToZero := GetVar("resetToZero", www, req)
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
	fmt.Fprintf(www, `{"status":"success","nPDFConverted":%d}`+"\n", nPDFConverted)
	return
}

// HandleExitServer - graceful server shutdown.
func HandleExitServer(www http.ResponseWriter, req *http.Request) {

	if !IsAuthKeyValid(www, req) {
		return
	}
	if isTLS {
		www.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
	}
	www.Header().Set("Content-Type", "application/json; charset=utf-8")

	//	// fmt.Printf("AT: %s - gCfg.AuthKey = [%s]\n", godebug.LF(), gCfg.AuthKey)
	//	found, auth_key := GetVar("auth_key", www, req)
	//	if gCfg.AuthKey != "" {
	//		// fmt.Printf("AT: %s - configed AuthKey [%s], found=%v ?auth_key=[%s]\n", godebug.LF(), gCfg.AuthKey, found, auth_key)
	//		if !found || auth_key != gCfg.AuthKey {
	//			// fmt.Printf("AT: %s\n", godebug.LF())
	//			www.WriteHeader(http.StatusUnauthorized) // 401
	//			return
	//		}
	//	}
	//	// fmt.Printf("AT: %s\n", godebug.LF())

	www.WriteHeader(http.StatusOK) // 200
	fmt.Fprintf(www, `{"status":"success","nPDFConverted":%d}`+"\n", nPDFConverted)

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
