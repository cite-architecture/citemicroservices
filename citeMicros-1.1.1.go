package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
)

type CTSURN struct {
	Stem      string
	Reference string
}

type Node struct {
	URN      []string `json:"urn"`
	Text     []string `json:"text,omitempty"`
	Previous []string `json:"previous"`
	Next     []string `json:"next"`
	Index    int      `json:"sequence"`
}

type Versions struct {
	Texts          string `json:"texts"`
	Textcatalog    string `json:"textcatalog,omitempty"`
	Citedata       string `json:"citedata,omitempty"`
	Citecatalog    string `json:"citecatalog,omitempty"`
	Citerelations  string `json:"citerelations,omitempty"`
	Citeextensions string `json:"citeextensions,omitempty"`
	DSE            string `json:"dse,omitempty"`
	ORCA           string `json:"orca,omitempty"`
}

type CITEResponse struct {
	Status   string   `json:"status"`
	Service  string   `json:"service"`
	Versions Versions `json:"versions"`
}

type VersionResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
	Version string `json:"version"`
}

type NodeResponse struct {
	RequestUrn []string `json:"requestUrn"`
	Status     string   `json:"status"`
	Service    string   `json:"service"`
	Message    string   `json:"message,omitempty"`
	URN        []string `json:"urns,omitempty"`
	Nodes      []Node   `json:""`
}

type URNResponse struct {
	RequestUrn []string `json:"requestUrn"`
	Status     string   `json:"status"`
	Service    string   `json:"service"`
	Message    string   `json:"message,omitempty"`
	URN        []string `json:"urns"`
}

type Work struct {
	WorkURN string
	URN     []string
	Text    []string
	Index   []int
}

type Collection struct {
	Works []Work
}

type CTSParams struct {
	Sourcetext string
}

type ServerConfig struct {
	Host       string `json:"host"`
	Port       string `json:"port"`
	Source     string `json:"cex_source"`
	TestSource string `json:"test_cex_source"`
}

func splitCTS(s string) CTSURN {
	var result CTSURN
	result = CTSURN{Stem: strings.Join(strings.Split(s, ":")[0:4], ":"), Reference: strings.Split(s, ":")[4]}
	return result
}

func LoadConfiguration(file string) ServerConfig {
	var config ServerConfig
	configFile, err := os.Open(file)
	defer configFile.Close()
	if err != nil {
		fmt.Println(err.Error())
	}
	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&config)
	return config
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func isRange(s string) bool {
	switch {
	case len(strings.Split(s, ":")) < 5:
		return false
	case strings.Contains(strings.Split(s, ":")[4], "-"):
		return true
	default:
		return false
	}
}

func isCTSURN(s string) bool {
	test := strings.Split(s, ":")
	switch {
	case len(test) < 4:
		return false
	case len(test) > 5:
		return false
	case test[0] != "urn":
		return false
	case test[1] != "cts":
		return false
	default:
		return true
	}
}

func boolcontains(s []bool, e bool) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func level1contains(s []string, e string) bool {
	var match []bool
	for i := range s {
		match2, _ := regexp.MatchString((e + "([:|.]*[0-9|a-z]+)$"), s[i])
		match = append(match, match2)
	}
	return boolcontains(match, true)
}

func level2contains(s []string, e string) bool {
	var match []bool
	for i := range s {
		match2, _ := regexp.MatchString((e + "([:|.]*[0-9|a-z]+).([0-9|a-z]+)$"), s[i])
		match = append(match, match2)
	}
	return boolcontains(match, true)
}

func level3contains(s []string, e string) bool {
	var match []bool
	for i := range s {
		match2, _ := regexp.MatchString((e + "([:|.]*[0-9|a-z]+).([0-9|a-z]+).([0-9|a-z]+)$"), s[i])
		match = append(match, match2)
	}
	return boolcontains(match, true)
}

func level4contains(s []string, e string) bool {
	var match []bool
	for i := range s {
		match2, _ := regexp.MatchString((e + "([:|.]*[0-9|a-z]+).([0-9|a-z]+).([0-9|a-z]+).([0-9|a-z]+)$"), s[i])
		match = append(match, match2)
	}
	return boolcontains(match, true)
}

func removeDuplicatesUnordered(elements []string) []string {
	encountered := map[string]bool{}

	// Create a map of all unique elements.
	for v := range elements {
		encountered[elements[v]] = true
	}

	// Place all keys from the map into a slice.
	result := []string{}
	for key, _ := range encountered {
		result = append(result, key)
	}
	return result
}

func main() {
	confvar := LoadConfiguration("./config.json")
	serverIP := confvar.Port
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/cite", ReturnCiteVersion)
	router.HandleFunc("/texts", ReturnWorkURNS)
	router.HandleFunc("/texts/version", ReturnTextsVersion)
	router.HandleFunc("/texts/first/{URN}", ReturnFirst)
	router.HandleFunc("/texts/last/{URN}", ReturnLast)
	router.HandleFunc("/texts/previous/{URN}", ReturnPrev)
	router.HandleFunc("/texts/next/{URN}", ReturnNext)
	router.HandleFunc("/texts/urns/{URN}", ReturnReff)
	router.HandleFunc("/texts/{URN}", ReturnPassage)
	router.HandleFunc("/{CEX}/texts/", ReturnWorkURNS)
	router.HandleFunc("/{CEX}/texts/first/{URN}", ReturnFirst)
	router.HandleFunc("/{CEX}/texts/last/{URN}", ReturnLast)
	router.HandleFunc("/{CEX}/texts/previous/{URN}", ReturnPrev)
	router.HandleFunc("/{CEX}/texts/next/{URN}", ReturnNext)
	router.HandleFunc("/{CEX}/texts/urns/{URN}", ReturnReff)
	router.HandleFunc("/{CEX}/texts/{URN}", ReturnPassage)
	router.HandleFunc("/", ReturnCiteVersion)
	headersOk := handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type"})
	originsOk := handlers.AllowedOrigins([]string{os.Getenv("ORIGIN_ALLOWED")})
	methodsOk := handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "PUT", "OPTIONS"})

	log.Println("Listening at" + serverIP + "...")
	log.Fatal(http.ListenAndServe(serverIP, handlers.CORS(originsOk, headersOk, methodsOk)(router)))
}

func getContent(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("GET error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Status error: %v", resp.StatusCode)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Read body: %v", err)
	}
	return data, nil
}

func ReturnWorkURNS(w http.ResponseWriter, r *http.Request) {
	confvar := LoadConfiguration("config.json")
	vars := mux.Vars(r)
	requestCEX := ""
	requestCEX = vars["CEX"]
	var sourcetext string
	switch {
	case requestCEX != "":
		sourcetext = confvar.Source + requestCEX + ".cex"
	default:
		sourcetext = confvar.TestSource
	}
	result := ParseURNS(CTSParams{Sourcetext: sourcetext})
	for i := range result.URN {
		result.URN[i] = strings.Join(strings.Split(result.URN[i], ":")[0:4], ":")
		result.URN[i] = result.URN[i] + ":"
	}
	result.URN = removeDuplicatesUnordered(result.URN)
	result.Service = "/texts"
	result.RequestUrn = []string{}
	resultJSON, _ := json.Marshal(result)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintln(w, string(resultJSON))
}

func ParseURNS(p CTSParams) URNResponse {
	input_file := p.Sourcetext
	data, err := getContent(input_file)
	if err != nil {
		return URNResponse{Status: "Exception", Message: "Couldn't open connection."}
	}

	str := string(data)
	// Remove comments
	str = strings.Split(str, "#!ctsdata")[1]
	str = strings.Split(str, "#!")[0]
	re := regexp.MustCompile("(?m)[\r\n]*^//.*$")
	str = re.ReplaceAllString(str, "")

	reader := csv.NewReader(strings.NewReader(str))
	reader.Comma = '#'
	reader.LazyQuotes = true
	reader.FieldsPerRecord = 2

	var response URNResponse

	for {
		line, error := reader.Read()
		if error == io.EOF {
			break
		} else if error != nil {
			log.Fatal(error)
		}
		response.URN = append(response.URN, line[0])
	}
	response.Status = "Success"
	return response
}

func ParseWork(p CTSParams) Work {
	input_file := p.Sourcetext
	data, err := getContent(input_file)
	if err != nil {
		return Work{}
	}

	str := string(data)
	str = strings.Split(str, "#!ctsdata")[1]
	str = strings.Split(str, "#!")[0]
	re := regexp.MustCompile("(?m)[\r\n]*^//.*$")
	str = re.ReplaceAllString(str, "")

	reader := csv.NewReader(strings.NewReader(str))
	reader.Comma = '#'
	reader.LazyQuotes = true
	reader.FieldsPerRecord = 2

	var response Work

	for {
		line, error := reader.Read()
		if error == io.EOF {
			break
		} else if error != nil {
			log.Fatal(error)
		}
		response.URN = append(response.URN, line[0])
		response.Text = append(response.Text, line[1])
	}
	return response
}

func ReturnCiteVersion(w http.ResponseWriter, r *http.Request) {
	var result CITEResponse
	result = CITEResponse{Status: "Success",
		Service:  "/cite",
		Versions: Versions{Texts: "1.1.0", Textcatalog: ""}}
	resultJSON, _ := json.Marshal(result)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintln(w, string(resultJSON))
}

func ReturnTextsVersion(w http.ResponseWriter, r *http.Request) {
	var result VersionResponse
	result = VersionResponse{
		Status:  "Success",
		Service: "/texts/version",
		Version: "1.1.0"}
	resultJSON, _ := json.Marshal(result)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintln(w, string(resultJSON))
}

func ReturnFirst(w http.ResponseWriter, r *http.Request) {
	confvar := LoadConfiguration("config.json")
	vars := mux.Vars(r)
	requestCEX := ""
	requestCEX = vars["CEX"]
	var sourcetext string
	switch {
	case requestCEX != "":
		sourcetext = confvar.Source + requestCEX + ".cex"
	default:
		sourcetext = confvar.TestSource
	}
	requestUrn := vars["URN"]
	if isCTSURN(requestUrn) != true {
		message := requestUrn + " is not valid CTS."
		result := NodeResponse{RequestUrn: []string{requestUrn}, Status: "Exception", Message: message}
		result.Service = "/texts/first"
		resultJSON, _ := json.Marshal(result)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		fmt.Fprintln(w, string(resultJSON))
		return
	}
	workResult := ParseWork(CTSParams{Sourcetext: sourcetext})
	works := append([]string(nil), workResult.URN...)
	for i := range workResult.URN {
		works[i] = strings.Join(strings.Split(workResult.URN[i], ":")[0:4], ":")
	}
	works = removeDuplicatesUnordered(works)
	workindex := 0
	for i := range works {
		if strings.Contains(requestUrn, works[i]) {
			teststring := works[i] + ":"
			switch {
			case requestUrn == works[i]:
				workindex = i + 1
			case strings.Contains(requestUrn, teststring):
				workindex = i + 1
			}
		}
	}
	var result NodeResponse
	switch {
	case workindex == 0:
		message := "No results for " + requestUrn
		result = NodeResponse{RequestUrn: []string{requestUrn}, Status: "Exception", Message: message}
	default:
		var RequestedWork Work
		RequestedWork.WorkURN = works[workindex-1]
		runindex := 0
		for i := range workResult.URN {
			if strings.Join(strings.Split(workResult.URN[i], ":")[0:4], ":") == RequestedWork.WorkURN {
				RequestedWork.URN = append(RequestedWork.URN, workResult.URN[i])
				RequestedWork.Text = append(RequestedWork.Text, workResult.Text[i])
				runindex++
				RequestedWork.Index = append(RequestedWork.Index, runindex)
			}
		}
		result = NodeResponse{RequestUrn: []string{requestUrn},
			Status: "Success",
			Nodes: []Node{Node{URN: []string{RequestedWork.URN[0]},
				Text:  []string{RequestedWork.Text[0]},
				Next:  []string{RequestedWork.URN[1]},
				Index: RequestedWork.Index[0]}}}
	}
	result.Service = "/texts/first"
	resultJSON, _ := json.Marshal(result)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintln(w, string(resultJSON))
}

func ReturnLast(w http.ResponseWriter, r *http.Request) {
	confvar := LoadConfiguration("config.json")
	vars := mux.Vars(r)
	requestCEX := ""
	requestCEX = vars["CEX"]
	var sourcetext string
	switch {
	case requestCEX != "":
		sourcetext = confvar.Source + requestCEX + ".cex"
	default:
		sourcetext = confvar.TestSource
	}
	requestUrn := vars["URN"]
	if isCTSURN(requestUrn) != true {
		message := requestUrn + " is not valid CTS."
		result := NodeResponse{RequestUrn: []string{requestUrn}, Status: "Exception", Message: message}
		result.Service = "/texts/last"
		resultJSON, _ := json.Marshal(result)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		fmt.Fprintln(w, string(resultJSON))
		return
	}
	workResult := ParseWork(CTSParams{Sourcetext: sourcetext})
	works := append([]string(nil), workResult.URN...)
	for i := range workResult.URN {
		works[i] = strings.Join(strings.Split(workResult.URN[i], ":")[0:4], ":")
	}
	works = removeDuplicatesUnordered(works)
	workindex := 0
	for i := range works {
		if strings.Contains(requestUrn, works[i]) {
			teststring := works[i] + ":"
			switch {
			case requestUrn == works[i]:
				workindex = i + 1
			case strings.Contains(requestUrn, teststring):
				workindex = i + 1
			}
		}
	}
	var result NodeResponse
	switch {
	case workindex == 0:
		message := "No results for " + requestUrn
		result = NodeResponse{RequestUrn: []string{requestUrn}, Status: "Exception", Message: message}
	default:
		var RequestedWork Work
		RequestedWork.WorkURN = works[workindex-1]
		runindex := 0
		for i := range workResult.URN {
			if strings.Join(strings.Split(workResult.URN[i], ":")[0:4], ":") == RequestedWork.WorkURN {
				RequestedWork.URN = append(RequestedWork.URN, workResult.URN[i])
				RequestedWork.Text = append(RequestedWork.Text, workResult.Text[i])
				runindex++
				RequestedWork.Index = append(RequestedWork.Index, runindex)
			}
		}
		result = NodeResponse{RequestUrn: []string{requestUrn},
			Status: "Success",
			Nodes: []Node{Node{URN: []string{RequestedWork.URN[len(RequestedWork.URN)-1]},
				Text:     []string{RequestedWork.Text[len(RequestedWork.URN)-1]},
				Previous: []string{RequestedWork.URN[len(RequestedWork.URN)-2]},
				Index:    RequestedWork.Index[len(RequestedWork.URN)-1]}}}
	}
	result.Service = "/texts/last"
	resultJSON, _ := json.Marshal(result)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintln(w, string(resultJSON))
}

func ReturnPrev(w http.ResponseWriter, r *http.Request) {
	confvar := LoadConfiguration("config.json")
	vars := mux.Vars(r)
	requestCEX := ""
	requestCEX = vars["CEX"]
	var sourcetext string
	switch {
	case requestCEX != "":
		sourcetext = confvar.Source + requestCEX + ".cex"
	default:
		sourcetext = confvar.TestSource
	}
	requestUrn := vars["URN"]
	if isCTSURN(requestUrn) != true {
		message := requestUrn + " is not valid CTS."
		result := NodeResponse{RequestUrn: []string{requestUrn}, Status: "Exception", Message: message}
		result.Service = "/texts/previous"
		resultJSON, _ := json.Marshal(result)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		fmt.Fprintln(w, string(resultJSON))
		return
	}
	workResult := ParseWork(CTSParams{Sourcetext: sourcetext})
	works := append([]string(nil), workResult.URN...)
	for i := range workResult.URN {
		works[i] = strings.Join(strings.Split(workResult.URN[i], ":")[0:4], ":")
	}
	works = removeDuplicatesUnordered(works)
	workindex := 0
	for i := range works {
		if strings.Contains(requestUrn, works[i]) {
			teststring := works[i] + ":"
			switch {
			case requestUrn == works[i]:
				workindex = i + 1
			case strings.Contains(requestUrn, teststring):
				workindex = i + 1
			}
		}
	}
	var result NodeResponse
	switch {
	case workindex == 0:
		message := "No results for " + requestUrn
		result = NodeResponse{RequestUrn: []string{requestUrn}, Status: "Exception", Message: message}
	default:
		var RequestedWork Work
		RequestedWork.WorkURN = works[workindex-1]
		runindex := 0
		for i := range workResult.URN {
			if strings.Join(strings.Split(workResult.URN[i], ":")[0:4], ":") == RequestedWork.WorkURN {
				RequestedWork.URN = append(RequestedWork.URN, workResult.URN[i])
				RequestedWork.Text = append(RequestedWork.Text, workResult.Text[i])
				runindex++
				RequestedWork.Index = append(RequestedWork.Index, runindex)
			}
		}
		var requestedIndex int
		for i := range RequestedWork.URN {
			if RequestedWork.URN[i] == requestUrn {
				requestedIndex = i
			}
		}
		switch {
		case contains(RequestedWork.URN, requestUrn):
			switch {
			case requestedIndex == 0:
				result = NodeResponse{RequestUrn: []string{requestUrn}, Status: "Success", Nodes: []Node{}}
			case requestedIndex-1 == 0:
				result = NodeResponse{RequestUrn: []string{requestUrn},
					Status: "Success",
					Nodes: []Node{Node{URN: []string{RequestedWork.URN[requestedIndex-1]},
						Text:  []string{RequestedWork.Text[requestedIndex-1]},
						Next:  []string{RequestedWork.URN[requestedIndex]},
						Index: RequestedWork.Index[requestedIndex-1]}}}
			default:
				result = NodeResponse{RequestUrn: []string{requestUrn},
					Status: "Success",
					Nodes: []Node{Node{URN: []string{RequestedWork.URN[requestedIndex-1]},
						Text:     []string{RequestedWork.Text[requestedIndex-1]},
						Next:     []string{RequestedWork.URN[requestedIndex]},
						Previous: []string{RequestedWork.URN[requestedIndex-2]},
						Index:    RequestedWork.Index[requestedIndex-1]}}}
			}
		default:
			message := "Could not find node to " + requestUrn + " in source."
			result = NodeResponse{RequestUrn: []string{requestUrn}, Status: "Exception", Message: message}
		}
	}
	result.Service = "/texts/previous"
	resultJSON, _ := json.Marshal(result)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintln(w, string(resultJSON))
}

func ReturnNext(w http.ResponseWriter, r *http.Request) {
	confvar := LoadConfiguration("config.json")
	vars := mux.Vars(r)
	requestCEX := ""
	requestCEX = vars["CEX"]
	var sourcetext string
	switch {
	case requestCEX != "":
		sourcetext = confvar.Source + requestCEX + ".cex"
	default:
		sourcetext = confvar.TestSource
	}
	requestUrn := vars["URN"]
	if isCTSURN(requestUrn) != true {
		message := requestUrn + " is not valid CTS."
		result := NodeResponse{RequestUrn: []string{requestUrn}, Status: "Exception", Message: message}
		result.Service = "/texts/next"
		resultJSON, _ := json.Marshal(result)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		fmt.Fprintln(w, string(resultJSON))
		return
	}
	workResult := ParseWork(CTSParams{Sourcetext: sourcetext})
	works := append([]string(nil), workResult.URN...)
	for i := range workResult.URN {
		works[i] = strings.Join(strings.Split(workResult.URN[i], ":")[0:4], ":")
	}
	works = removeDuplicatesUnordered(works)
	workindex := 0
	for i := range works {
		if strings.Contains(requestUrn, works[i]) {
			teststring := works[i] + ":"
			switch {
			case requestUrn == works[i]:
				workindex = i + 1
			case strings.Contains(requestUrn, teststring):
				workindex = i + 1
			}
		}
	}
	var result NodeResponse
	switch {
	case workindex == 0:
		message := "No results for " + requestUrn
		result = NodeResponse{RequestUrn: []string{requestUrn}, Status: "Exception", Message: message}
	default:
		var RequestedWork Work
		RequestedWork.WorkURN = works[workindex-1]
		runindex := 0
		for i := range workResult.URN {
			if strings.Join(strings.Split(workResult.URN[i], ":")[0:4], ":") == RequestedWork.WorkURN {
				RequestedWork.URN = append(RequestedWork.URN, workResult.URN[i])
				RequestedWork.Text = append(RequestedWork.Text, workResult.Text[i])
				runindex++
				RequestedWork.Index = append(RequestedWork.Index, runindex)
			}
		}
		var requestedIndex int
		for i := range RequestedWork.URN {
			if RequestedWork.URN[i] == requestUrn {
				requestedIndex = i
			}
		}
		switch {
		case contains(RequestedWork.URN, requestUrn):
			switch {
			case requestedIndex == len(RequestedWork.URN)-1:
				result = NodeResponse{RequestUrn: []string{requestUrn}, Status: "Success", Nodes: []Node{}}
			case requestedIndex+1 == len(RequestedWork.URN)-1:
				result = NodeResponse{RequestUrn: []string{requestUrn},
					Status: "Success",
					Nodes: []Node{Node{URN: []string{RequestedWork.URN[requestedIndex+1]},
						Text:     []string{RequestedWork.Text[requestedIndex+1]},
						Previous: []string{RequestedWork.URN[requestedIndex]},
						Index:    RequestedWork.Index[requestedIndex+1]}}}
			default:
				result = NodeResponse{RequestUrn: []string{requestUrn},
					Status: "Success",
					Nodes: []Node{Node{URN: []string{RequestedWork.URN[requestedIndex+1]},
						Text:     []string{RequestedWork.Text[requestedIndex+1]},
						Next:     []string{RequestedWork.URN[requestedIndex+2]},
						Previous: []string{RequestedWork.URN[requestedIndex]},
						Index:    RequestedWork.Index[requestedIndex+1]}}}
			}
		default:
			message := "Could not find node to " + requestUrn + " in source."
			result = NodeResponse{RequestUrn: []string{requestUrn}, Status: "Exception", Message: message}
		}
	}
	result.Service = "/texts/next"
	resultJSON, _ := json.Marshal(result)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintln(w, string(resultJSON))
}

func ReturnReff(w http.ResponseWriter, r *http.Request) {
	confvar := LoadConfiguration("config.json")
	vars := mux.Vars(r)
	requestCEX := ""
	requestCEX = vars["CEX"]
	var sourcetext string
	switch {
	case requestCEX != "":
		sourcetext = confvar.Source + requestCEX + ".cex"
	default:
		sourcetext = confvar.TestSource
	}
	requestUrn := vars["URN"]
	if isCTSURN(requestUrn) != true {
		message := requestUrn + " is not valid CTS."
		result := NodeResponse{RequestUrn: []string{requestUrn}, Status: "Exception", Message: message}
		result.Service = "/texts/urns"
		resultJSON, _ := json.Marshal(result)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		fmt.Fprintln(w, string(resultJSON))
		return
	}
	workResult := ParseWork(CTSParams{Sourcetext: sourcetext})
	works := append([]string(nil), workResult.URN...)
	for i := range workResult.URN {
		works[i] = strings.Join(strings.Split(workResult.URN[i], ":")[0:4], ":")
	}
	works = removeDuplicatesUnordered(works)
	workindex := 0
	for i := range works {
		if strings.Contains(requestUrn, works[i]) {
			teststring := works[i] + ":"
			switch {
			case requestUrn == works[i]:
				workindex = i + 1
			case strings.Contains(requestUrn, teststring):
				workindex = i + 1
			}
		}
	}
	var result URNResponse
	switch {
	case workindex == 0:
		message := "No results for " + requestUrn
		result = URNResponse{RequestUrn: []string{requestUrn}, Status: "Exception", Message: message}
		result.Service = "/texts/urns"
		resultJSON, _ := json.Marshal(result)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		fmt.Fprintln(w, string(resultJSON))
	default:
		var RequestedWork Work
		RequestedWork.WorkURN = works[workindex-1]
		runindex := 0
		for i := range workResult.URN {
			if strings.Join(strings.Split(workResult.URN[i], ":")[0:4], ":") == RequestedWork.WorkURN {
				RequestedWork.URN = append(RequestedWork.URN, workResult.URN[i])
				RequestedWork.Text = append(RequestedWork.Text, workResult.Text[i])
				runindex++
				RequestedWork.Index = append(RequestedWork.Index, runindex)
			}
		}
		switch {
		case isRange(requestUrn):
			ctsurn := splitCTS(requestUrn)
			ctsrange := strings.Split(ctsurn.Reference, "-")
			startURN := ctsurn.Stem + ":" + ctsrange[0]
			endURN := ctsurn.Stem + ":" + ctsrange[1]
			var startindex, endindex int
			switch {
			case contains(RequestedWork.URN, startURN):
				for i := range RequestedWork.URN {
					if RequestedWork.URN[i] == startURN {
						startindex = i
					}
				}
			case level1contains(RequestedWork.URN, startURN):
				var match []bool
				for i := range RequestedWork.URN {
					match2, _ := regexp.MatchString((startURN + "([:|.]*[0-9|a-z]+)$"), RequestedWork.URN[i])
					match = append(match, match2)
				}
				for i := range match {
					if match[i] == true {
						startindex = i
						break
					}
				}
			case level2contains(RequestedWork.URN, startURN):
				var match []bool
				for i := range RequestedWork.URN {
					match2, _ := regexp.MatchString((startURN + "([:|.]*[0-9|a-z]+).([0-9|a-z]+)$"), RequestedWork.URN[i])
					match = append(match, match2)
				}
				for i := range match {
					if match[i] == true {
						startindex = i
						break
					}
				}
			case level3contains(RequestedWork.URN, startURN):
				var match []bool
				for i := range RequestedWork.URN {
					match2, _ := regexp.MatchString((startURN + "([:|.]*[0-9|a-z]+).([0-9|a-z]+).([0-9|a-z]+)$"), RequestedWork.URN[i])
					match = append(match, match2)
				}
				for i := range match {
					if match[i] == true {
						startindex = i
						break
					}
				}
			case level4contains(RequestedWork.URN, startURN):
				var match []bool
				for i := range RequestedWork.URN {
					match2, _ := regexp.MatchString((startURN + "([:|.]*[0-9|a-z]+).([0-9|a-z]+).([0-9|a-z]+).([0-9|a-z]+)$"), RequestedWork.URN[i])
					match = append(match, match2)
				}
				for i := range match {
					if match[i] == true {
						startindex = i
						break
					}
				}
			default:
				startindex = 0
			}
			switch {
			case contains(RequestedWork.URN, endURN):
				for i := range RequestedWork.URN {
					if RequestedWork.URN[i] == endURN {
						endindex = i
					}
				}
			case level1contains(RequestedWork.URN, endURN):
				var match []bool
				for i := range RequestedWork.URN {
					match2, _ := regexp.MatchString((endURN + "([:|.]*[0-9|a-z]+)$"), RequestedWork.URN[i])
					match = append(match, match2)
				}
				for i := len(match) - 1; i >= 0; i-- {
					if match[i] == true {
						endindex = i
						break
					}
				}
			case level2contains(RequestedWork.URN, endURN):
				var match []bool
				for i := range RequestedWork.URN {
					match2, _ := regexp.MatchString((endURN + "([:|.]*[0-9|a-z]+).([0-9|a-z]+)$"), RequestedWork.URN[i])
					match = append(match, match2)
				}
				for i := len(match) - 1; i >= 0; i-- {
					if match[i] == true {
						endindex = i
						break
					}
				}
			case level3contains(RequestedWork.URN, endURN):
				var match []bool
				for i := range RequestedWork.URN {
					match2, _ := regexp.MatchString((endURN + "([:|.]*[0-9|a-z]+).([0-9|a-z]+).([0-9|a-z]+)$"), RequestedWork.URN[i])
					match = append(match, match2)
				}
				for i := len(match) - 1; i >= 0; i-- {
					if match[i] == true {
						endindex = i
						break
					}
				}
			case level4contains(RequestedWork.URN, endURN):
				var match []bool
				for i := range RequestedWork.URN {
					match2, _ := regexp.MatchString((endURN + "([:|.]*[0-9|a-z]+).([0-9|a-z]+).([0-9|a-z]+).([0-9|a-z]+)$"), RequestedWork.URN[i])
					match = append(match, match2)
				}
				for i := len(match) - 1; i >= 0; i-- {
					if match[i] == true {
						endindex = i
						break
					}
				}
			default:
				endindex = len(RequestedWork.URN) - 1
			}
			range_urn := RequestedWork.URN[startindex : endindex+1]
			result = URNResponse{RequestUrn: []string{requestUrn}, Status: "Success", URN: range_urn}
			result.Service = "/texts/urns"
			resultJSON, _ := json.Marshal(result)
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			fmt.Fprintln(w, string(resultJSON))
		default:
			switch {
			case contains(RequestedWork.URN, requestUrn):
				result = URNResponse{RequestUrn: []string{requestUrn}, Status: "Success", URN: []string{requestUrn}}
			case level1contains(RequestedWork.URN, requestUrn):
				var matchingURNs []string
				var match []bool
				for i := range RequestedWork.URN {
					match2, _ := regexp.MatchString((requestUrn + "([:|.]*[0-9|a-z]+)$"), RequestedWork.URN[i])
					match = append(match, match2)
				}
				for i := range match {
					if match[i] == true {
						matchingURNs = append(matchingURNs, RequestedWork.URN[i])
					}
				}
				result = URNResponse{RequestUrn: []string{requestUrn}, Status: "Success", URN: matchingURNs}
			case level2contains(RequestedWork.URN, requestUrn):
				var matchingURNs []string
				var match []bool
				for i := range RequestedWork.URN {
					match2, _ := regexp.MatchString((requestUrn + "([:|.]*[0-9|a-z]+).([0-9|a-z]+)$"), RequestedWork.URN[i])
					match = append(match, match2)
				}
				for i := range match {
					if match[i] == true {
						matchingURNs = append(matchingURNs, RequestedWork.URN[i])
					}
				}
				result = URNResponse{RequestUrn: []string{requestUrn}, Status: "Success", URN: matchingURNs}
			case level3contains(RequestedWork.URN, requestUrn):
				var matchingURNs []string
				var match []bool
				for i := range RequestedWork.URN {
					match2, _ := regexp.MatchString((requestUrn + "([:|.]*[0-9|a-z]+).([0-9|a-z]+).([0-9|a-z]+)$"), RequestedWork.URN[i])
					match = append(match, match2)
				}
				for i := range match {
					if match[i] == true {
						matchingURNs = append(matchingURNs, RequestedWork.URN[i])
					}
				}
				result = URNResponse{RequestUrn: []string{requestUrn}, Status: "Success", URN: matchingURNs}
			case level4contains(RequestedWork.URN, requestUrn):
				var matchingURNs []string
				var match []bool
				for i := range RequestedWork.URN {
					match2, _ := regexp.MatchString((requestUrn + "([:|.]*[0-9|a-z]+).([0-9|a-z]+).([0-9|a-z]+)$"), RequestedWork.URN[i])
					match = append(match, match2)
				}
				for i := range match {
					if match[i] == true {
						matchingURNs = append(matchingURNs, RequestedWork.URN[i])
					}
				}
				result = URNResponse{RequestUrn: []string{requestUrn}, Status: "Success", URN: matchingURNs}
			default:
				result = URNResponse{RequestUrn: []string{requestUrn}, Status: "Exception", Message: "Couldn't find URN."}
			}
			result.Service = "/texts/urns"
			resultJSON, _ := json.Marshal(result)
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			fmt.Fprintln(w, string(resultJSON))
		}
	}
}

func ReturnPassage(w http.ResponseWriter, r *http.Request) {
	confvar := LoadConfiguration("config.json")
	vars := mux.Vars(r)
	requestCEX := ""
	requestCEX = vars["CEX"]
	var sourcetext string
	switch {
	case requestCEX != "":
		sourcetext = confvar.Source + requestCEX + ".cex"
	default:
		sourcetext = confvar.TestSource
	}
	requestUrn := vars["URN"]
	if isCTSURN(requestUrn) != true {
		message := requestUrn + " is not valid CTS."
		result := NodeResponse{RequestUrn: []string{requestUrn}, Status: "Exception", Message: message}
		result.Service = "/texts"
		resultJSON, _ := json.Marshal(result)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		fmt.Fprintln(w, string(resultJSON))
		return
	}
	workResult := ParseWork(CTSParams{Sourcetext: sourcetext})
	works := append([]string(nil), workResult.URN...)
	for i := range workResult.URN {
		works[i] = strings.Join(strings.Split(workResult.URN[i], ":")[0:4], ":")
	}
	works = removeDuplicatesUnordered(works)
	workindex := 0
	for i := range works {
		if strings.Contains(requestUrn, works[i]) {
			teststring := works[i] + ":"
			switch {
			case requestUrn == works[i]:
				workindex = i + 1
			case strings.Contains(requestUrn, teststring):
				workindex = i + 1
			}
		}
	}
	var result NodeResponse
	switch {
	case workindex == 0:
		message := "No results for " + requestUrn
		result = NodeResponse{RequestUrn: []string{requestUrn}, Status: "Exception", Message: message}
	default:
		var RequestedWork Work
		RequestedWork.WorkURN = works[workindex-1]
		runindex := 0
		for i := range workResult.URN {
			if strings.Join(strings.Split(workResult.URN[i], ":")[0:4], ":") == RequestedWork.WorkURN {
				RequestedWork.URN = append(RequestedWork.URN, workResult.URN[i])
				RequestedWork.Text = append(RequestedWork.Text, workResult.Text[i])
				runindex++
				RequestedWork.Index = append(RequestedWork.Index, runindex)
			}
		}
		var requestedIndex int
		for i := range RequestedWork.URN {
			if RequestedWork.URN[i] == requestUrn {
				requestedIndex = i
			}
		}
		switch {
		case contains(RequestedWork.URN, requestUrn):
			switch {
			case requestedIndex == 0:
				result = NodeResponse{RequestUrn: []string{requestUrn},
					Status: "Success",
					Nodes: []Node{Node{URN: []string{RequestedWork.URN[requestedIndex]},
						Text:  []string{RequestedWork.Text[requestedIndex]},
						Next:  []string{RequestedWork.URN[requestedIndex+1]},
						Index: RequestedWork.Index[requestedIndex]}}}
			case requestedIndex == len(RequestedWork.URN)-1:
				result = NodeResponse{RequestUrn: []string{requestUrn},
					Status: "Success",
					Nodes: []Node{Node{URN: []string{RequestedWork.URN[requestedIndex]},
						Text:     []string{RequestedWork.Text[requestedIndex]},
						Previous: []string{RequestedWork.URN[requestedIndex-1]},
						Index:    RequestedWork.Index[requestedIndex]}}}
			default:
				result = NodeResponse{RequestUrn: []string{requestUrn},
					Status: "Success",
					Nodes: []Node{Node{URN: []string{RequestedWork.URN[requestedIndex]},
						Text:     []string{RequestedWork.Text[requestedIndex]},
						Next:     []string{RequestedWork.URN[requestedIndex+1]},
						Previous: []string{RequestedWork.URN[requestedIndex-1]},
						Index:    RequestedWork.Index[requestedIndex]}}}
			}
		case level1contains(RequestedWork.URN, requestUrn):
			var matchingNodes []Node
			var match []bool
			for i := range RequestedWork.URN {
				match2, _ := regexp.MatchString((requestUrn + "([:|.]*[0-9|a-z]+)$"), RequestedWork.URN[i])
				match = append(match, match2)
			}
			for i := range match {
				if match[i] == true {
					previousnode := ""
					nextnode := ""
					if RequestedWork.Index[i] > 1 {
						previousnode = RequestedWork.URN[RequestedWork.Index[i]-2]
					}
					if RequestedWork.Index[i] < len(RequestedWork.URN) {
						nextnode = RequestedWork.URN[RequestedWork.Index[i]]
					}
					matchingNodes = append(matchingNodes, Node{URN: []string{RequestedWork.URN[i]}, Text: []string{RequestedWork.Text[i]}, Previous: []string{previousnode}, Next: []string{nextnode}, Index: RequestedWork.Index[i]})
				}
			}
			result = NodeResponse{RequestUrn: []string{requestUrn}, Status: "Success", Nodes: matchingNodes}
		case level2contains(RequestedWork.URN, requestUrn):
			var matchingNodes []Node
			var match []bool
			for i := range RequestedWork.URN {
				match2, _ := regexp.MatchString((requestUrn + "([:|.]*[0-9|a-z]+).([0-9|a-z]+)$"), RequestedWork.URN[i])
				match = append(match, match2)
			}
			for i := range match {
				if match[i] == true {
					previousnode := ""
					nextnode := ""
					if RequestedWork.Index[i] > 1 {
						previousnode = RequestedWork.URN[RequestedWork.Index[i]-2]
					}
					if RequestedWork.Index[i] < len(RequestedWork.URN) {
						nextnode = RequestedWork.URN[RequestedWork.Index[i]]
					}
					matchingNodes = append(matchingNodes, Node{URN: []string{RequestedWork.URN[i]}, Text: []string{RequestedWork.Text[i]}, Previous: []string{previousnode}, Next: []string{nextnode}, Index: RequestedWork.Index[i]})
				}
			}
			result = NodeResponse{RequestUrn: []string{requestUrn}, Status: "Success", Nodes: matchingNodes}
		case level3contains(RequestedWork.URN, requestUrn):
			var matchingNodes []Node
			var match []bool
			for i := range RequestedWork.URN {
				match2, _ := regexp.MatchString((requestUrn + "([:|.]*[0-9|a-z]+).([0-9|a-z]+).([0-9|a-z]+)$"), RequestedWork.URN[i])
				match = append(match, match2)
			}
			for i := range match {
				if match[i] == true {
					previousnode := ""
					nextnode := ""
					if RequestedWork.Index[i] > 1 {
						previousnode = RequestedWork.URN[RequestedWork.Index[i]-2]
					}
					if RequestedWork.Index[i] < len(RequestedWork.URN) {
						nextnode = RequestedWork.URN[RequestedWork.Index[i]]
					}
					matchingNodes = append(matchingNodes, Node{URN: []string{RequestedWork.URN[i]}, Text: []string{RequestedWork.Text[i]}, Previous: []string{previousnode}, Next: []string{nextnode}, Index: RequestedWork.Index[i]})
				}
			}
			result = NodeResponse{RequestUrn: []string{requestUrn}, Status: "Success", Nodes: matchingNodes}
		case level4contains(RequestedWork.URN, requestUrn):
			var matchingNodes []Node
			var match []bool
			for i := range RequestedWork.URN {
				match2, _ := regexp.MatchString((requestUrn + "([:|.]*[0-9|a-z]+).([0-9|a-z]+).([0-9|a-z]+).([0-9|a-z]+)$"), RequestedWork.URN[i])
				match = append(match, match2)
			}
			for i := range match {
				if match[i] == true {
					previousnode := ""
					nextnode := ""
					if RequestedWork.Index[i] > 1 {
						previousnode = RequestedWork.URN[RequestedWork.Index[i]-2]
					}
					if RequestedWork.Index[i] < len(RequestedWork.URN) {
						nextnode = RequestedWork.URN[RequestedWork.Index[i]]
					}
					matchingNodes = append(matchingNodes, Node{URN: []string{RequestedWork.URN[i]}, Text: []string{RequestedWork.Text[i]}, Previous: []string{previousnode}, Next: []string{nextnode}, Index: RequestedWork.Index[i]})
				}
			}
			result = NodeResponse{RequestUrn: []string{requestUrn}, Status: "Success", Nodes: matchingNodes}
		case isRange(requestUrn):
			var rangeNodes []Node
			ctsurn := splitCTS(requestUrn)
			ctsrange := strings.Split(ctsurn.Reference, "-")
			startURN := ctsurn.Stem + ":" + ctsrange[0]
			endURN := ctsurn.Stem + ":" + ctsrange[1]
			var startindex, endindex int
			switch {
			case contains(RequestedWork.URN, startURN):
				for i := range RequestedWork.URN {
					if RequestedWork.URN[i] == startURN {
						startindex = i
					}
				}
			case level1contains(RequestedWork.URN, startURN):
				var match []bool
				for i := range RequestedWork.URN {
					match2, _ := regexp.MatchString((startURN + "([:|.]*[0-9|a-z]+)$"), RequestedWork.URN[i])
					match = append(match, match2)
				}
				for i := range match {
					if match[i] == true {
						startindex = i
						break
					}
				}
			case level2contains(RequestedWork.URN, startURN):
				var match []bool
				for i := range RequestedWork.URN {
					match2, _ := regexp.MatchString((startURN + "([:|.]*[0-9|a-z]+).([0-9|a-z]+)$"), RequestedWork.URN[i])
					match = append(match, match2)
				}
				for i := range match {
					if match[i] == true {
						startindex = i
						break
					}
				}
			case level3contains(RequestedWork.URN, startURN):
				var match []bool
				for i := range RequestedWork.URN {
					match2, _ := regexp.MatchString((startURN + "([:|.]*[0-9|a-z]+).([0-9|a-z]+).([0-9|a-z]+)$"), RequestedWork.URN[i])
					match = append(match, match2)
				}
				for i := range match {
					if match[i] == true {
						startindex = i
						break
					}
				}
			case level4contains(RequestedWork.URN, startURN):
				var match []bool
				for i := range RequestedWork.URN {
					match2, _ := regexp.MatchString((startURN + "([:|.]*[0-9|a-z]+).([0-9|a-z]+).([0-9|a-z]+).([0-9|a-z]+)$"), RequestedWork.URN[i])
					match = append(match, match2)
				}
				for i := range match {
					if match[i] == true {
						startindex = i
						break
					}
				}
			default:
				startindex = 0
			}
			switch {
			case contains(RequestedWork.URN, endURN):
				for i := range RequestedWork.URN {
					if RequestedWork.URN[i] == endURN {
						endindex = i
					}
				}
			case level1contains(RequestedWork.URN, endURN):
				var match []bool
				for i := range RequestedWork.URN {
					match2, _ := regexp.MatchString((endURN + "([:|.]*[0-9|a-z]+)$"), RequestedWork.URN[i])
					match = append(match, match2)
				}
				for i := len(match) - 1; i >= 0; i-- {
					if match[i] == true {
						endindex = i
						break
					}
				}
			case level2contains(RequestedWork.URN, endURN):
				var match []bool
				for i := range RequestedWork.URN {
					match2, _ := regexp.MatchString((endURN + "([:|.]*[0-9|a-z]+).([0-9|a-z]+)$"), RequestedWork.URN[i])
					match = append(match, match2)
				}
				for i := len(match) - 1; i >= 0; i-- {
					if match[i] == true {
						endindex = i
						break
					}
				}
			case level3contains(RequestedWork.URN, endURN):
				var match []bool
				for i := range RequestedWork.URN {
					match2, _ := regexp.MatchString((endURN + "([:|.]*[0-9|a-z]+).([0-9|a-z]+).([0-9|a-z]+)$"), RequestedWork.URN[i])
					match = append(match, match2)
				}
				for i := len(match) - 1; i >= 0; i-- {
					if match[i] == true {
						endindex = i
						break
					}
				}
			case level4contains(RequestedWork.URN, endURN):
				var match []bool
				for i := range RequestedWork.URN {
					match2, _ := regexp.MatchString((endURN + "([:|.]*[0-9|a-z]+).([0-9|a-z]+).([0-9|a-z]+).([0-9|a-z]+)$"), RequestedWork.URN[i])
					match = append(match, match2)
				}
				for i := len(match) - 1; i >= 0; i-- {
					if match[i] == true {
						endindex = i
						break
					}
				}
			default:
				endindex = len(RequestedWork.URN) - 1
			}
			range_urn := RequestedWork.URN[startindex : endindex+1]
			range_text := RequestedWork.Text[startindex : endindex+1]
			range_index := RequestedWork.Index[startindex : endindex+1]
			for i := range range_urn {
				previousnode := ""
				nextnode := ""
				if range_index[i] > 1 {
					previousnode = RequestedWork.URN[range_index[i]-2]
				}
				if range_index[i] < len(RequestedWork.URN) {
					nextnode = RequestedWork.URN[range_index[i]]
				}
				rangeNodes = append(rangeNodes, Node{URN: []string{range_urn[i]}, Text: []string{range_text[i]}, Previous: []string{previousnode}, Next: []string{nextnode}, Index: range_index[i]})
			}
			result = NodeResponse{RequestUrn: []string{requestUrn}, Status: "Success", Nodes: rangeNodes}
		default:
			message := "Could not find node to " + requestUrn + " in source."
			result = NodeResponse{RequestUrn: []string{requestUrn}, Status: "Exception", Message: message}
		}
	}
	result.Service = "/texts"
	resultJSON, _ := json.Marshal(result)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintln(w, string(resultJSON))
}
