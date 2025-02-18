package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"frc/6510"
	"frc/6510/internal/model"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
)

var drivers = template.Must(template.ParseFS(root.Templates, "**/*.tmpl"))

func main() {
	model.Init()
	re, err := regexp.Compile(`-\?[^0-9]`)
	if err != nil {
		log.Fatal(err)
	}
	numberRe = re

	libFS, err := fs.Sub(root.JSLib, "content")
	if err != nil {
		log.Fatal(err)
	}

	router := http.NewServeMux()
	router.Handle("GET /", http.FileServerFS(libFS))

	router.HandleFunc("POST /matches/validate-new", validateMatchForm)

	router.HandleFunc("POST /matches/new", newMatch)

	router.HandleFunc("GET /match/{hash}", match)
	router.HandleFunc("POST /match/{hash}", updateMatch)
	router.HandleFunc("POST /delete/{hash}", delete)

	router.HandleFunc("GET /{$}", index)

	log.Println("Server started at http://localhost:8080")
	if err := open("http://localhost:8080"); err != nil {
		log.Println("Failed to open browser: " + err.Error())
	}
	http.ListenAndServe(":8080", preventCaching(router))
}

// open opens the specified URL in the default browser of the user.
func open(url string) error {
    var cmd string
    var args []string

    switch runtime.GOOS {
    case "windows":
        cmd = "cmd"
        args = []string{"/c", "start"}
    case "darwin":
        cmd = "open"
    default: // "linux", "freebsd", "openbsd", "netbsd"
        cmd = "xdg-open"
    }
    args = append(args, url)
    return exec.Command(cmd, args...).Start()
}

func preventCaching(delegate http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Cache-Control", "no-cache")
		w.Header().Add("Cache-Control", "no-store")
		w.Header().Add("Cache-Control", "must-revalidate")
		w.Header().Add("Cache-Control", "max-age=0")
		delegate.ServeHTTP(w, r)
	}
}

func index(w http.ResponseWriter, r *http.Request) {
	data := model.Page[model.Index]{
		Title: "Pymble Pioneer",
		Data: model.Index{
			Form: model.MatchForm{
				Meta: model.Meta{
					ScouterName: "",
					MatchNumber: 0,
					TeamNumber:  0,
					MatchType:   model.Practice,
				},
				Error: "",
			},
		},
	}
	if err := filepath.WalkDir(model.Dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type().IsDir() {
			if d.Name() == "pymble-pioneer" {
				return nil
			}
			return filepath.SkipDir
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		decoder := json.NewDecoder(file)
		match := model.Match{}
		err = decoder.Decode(&match)
		if err != nil {
			err = os.Remove(model.Dir)
			if err != nil {
				return err
			}
		}

		hash := match.Meta.Hash()
		filename := fmt.Sprintf("%x", hash)
		if d.Name() != filename {
			if err = os.Rename(path, filepath.Join(filepath.Dir(path), filename)); err != nil {
				return err
			}
		}

		data.Data.Matches = append(data.Data.Matches, match)

		return nil
	}); err != nil {
		log.Fatal(err)
	}

	drivers.ExecuteTemplate(w, "index.html.tmpl", data)
}

func validateMatchForm(w http.ResponseWriter, r *http.Request) {
	form, _ := validateNewMatch(r)
	drivers.ExecuteTemplate(w, "new-match-form.html.tmpl", form)
}

var numberRe *regexp.Regexp

func validateNewMatch(r *http.Request) (model.MatchForm, []byte) {
	form := model.MatchForm{}
	scouterName := r.PostFormValue("scouterName")
	matchNumber, err := validatePostFormFieldNumber("matchNumber", r)
	if err != nil {
		form.Error += err.Error() + "\n"
	}
	teamNumber, err := validatePostFormFieldNumber("teamNumber", r)
	if err != nil {
		form.Error += err.Error() + "\n"
	}
	matchType, exists := model.MatchTypeMap[r.PostFormValue("matchType")]
	if !exists {
		matchType = model.Practice
		form.Error += "Invalid Match Type\n"
	}
	form.Meta = model.Meta{
		ScouterName: scouterName,
		MatchNumber: matchNumber,
		TeamNumber:  teamNumber,
		MatchType:   matchType,
	}
	hash := form.Meta.Hash()
	if _, err := os.Stat(filepath.Join(model.Dir, fmt.Sprintf("%x", hash))); err == nil {
		form.Error += "Record already exists\n"
	} else if !errors.Is(err, fs.ErrNotExist) {
		log.Fatal(err)
	}

	return form, hash
}

func newMatch(w http.ResponseWriter, r *http.Request) {
	form, hash := validateNewMatch(r)
	if form.Error != "" {
		drivers.ExecuteTemplate(w, "new-match-form.html.tmpl", form)
	} else {
		file, err := os.Create(filepath.Join(model.Dir, fmt.Sprintf("%x", hash)))
		if err != nil {
			log.Fatal(err)
		}
		jsonEncoder := json.NewEncoder(file)
		err = jsonEncoder.Encode(model.Match{
			Meta: form.Meta,
		})
		if err != nil {
			log.Fatal(err)
		}
		if err := file.Close(); err != nil {
			log.Fatal(err)
		}
		w.Header()["HX-Redirect"] = []string{fmt.Sprintf("/match/%x", hash)}
		w.WriteHeader(200)
	}
}

func match(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("hash")
	log.Printf("requested: %s", hash)
	filename, err := filepath.Abs(filepath.Join(model.Dir, hash))
	if err != nil {
		log.Fatal(err)
	}
	file, err := os.Open(filename)
	if errors.Is(err, os.ErrNotExist) {
		http.NotFound(w, r)
		return
	} else if err != nil {
		log.Fatal(err)
	}
	match := model.Match{}
	decoder := json.NewDecoder(file)
	if err = decoder.Decode(&match); err != nil {
		log.Println(err)
		log.Printf("encountered error parsing %s, attempting to delete it\n", filename)
		os.Remove(filename)
		return
	}

	if err = drivers.ExecuteTemplate(w, "match.html.tmpl", model.Page[model.Match]{
		Title: fmt.Sprintf("%s Editing Team %d - %s Match %d", match.Meta.ScouterName, match.Meta.TeamNumber, match.Meta.MatchType, match.Meta.MatchNumber),
		Data:  match,
	}); err != nil {
		log.Println(err)
	}
}

func validatePostFormFieldNumber(fieldName string, r *http.Request) (uint64, error) {
	return validateNumber(fieldName, r.PostFormValue(fieldName))
}

func validateNumber(fieldName string, str string) (uint64, error) {
	str = numberRe.ReplaceAllString(str, "")
	var number uint64
	if str != "" {
		var err error
		number, err = strconv.ParseUint(str, 10, 64)
		if err != nil {
			return 0, errors.New(fmt.Sprintf("Invalid %s", fieldName))
		}
	}
	return number, nil
}

func updateMatch(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("hash")
	log.Printf("edited: %s", hash)
	filename, err := filepath.Abs(filepath.Join(model.Dir, hash))
	if err != nil {
		log.Fatal(err)
	}
	file, err := os.Open(filename)
	if errors.Is(err, os.ErrNotExist) {
		http.NotFound(w, r)
		return
	} else if err != nil {
		log.Fatal(err)
	}
	match := model.Match{}
	decoder := json.NewDecoder(file)
	if err = decoder.Decode(&match); err != nil {
		log.Println(err)
		log.Printf("encountered error parsing %s, attempting to delete it\n", filename)
		os.Remove(filename)
		return
	}

	minorFoul, _ := validatePostFormFieldNumber("minorFoul", r)
	match.Sticky.Fouls.Minor = minorFoul
	majorFoul, _ := validatePostFormFieldNumber("majorFoul", r)
	match.Sticky.Fouls.Major = majorFoul
	yellowCard, _ := validatePostFormFieldNumber("yellowCard", r)
	match.Sticky.Fouls.YellowCard = yellowCard
	redCard, _ := validatePostFormFieldNumber("redCard", r)
	match.Sticky.Fouls.RedCard = redCard

	coopertition := r.PostFormValue("coopertition")
	match.Sticky.Coopertition = coopertition != ""

	match.Sticky.Comment = r.PostFormValue("comment")

	startingPosition, _ := validatePostFormFieldNumber("startingPosition", r)
	if startingPosition > 3 { startingPosition = 0 }
	match.Prematch.StartingPosition = model.Position(startingPosition)

	driverStation, _ := validatePostFormFieldNumber("driverStation", r)
	if driverStation > 3 { driverStation = 0 }
	match.Prematch.DriverStation = model.Position(driverStation)

	crossedLine := r.PostFormValue("crossedLine")
	match.Auto.CrossedLine = crossedLine != ""

	autoL4, _ := validatePostFormFieldNumber("autoL4", r)
	match.Auto.L4 = autoL4
	autoL3, _ := validatePostFormFieldNumber("autoL3", r)
	match.Auto.L3 = autoL3
	autoL2, _ := validatePostFormFieldNumber("autoL2", r)
	match.Auto.L2 = autoL2
	autoL1, _ := validatePostFormFieldNumber("autoL1", r)
	match.Auto.L1 = autoL1

	autoProcessor, _ := validatePostFormFieldNumber("autoProcessor", r)
	match.Auto.Processor = autoProcessor
	autoRemoved, _ := validatePostFormFieldNumber("autoRemoved", r)
	match.Auto.Removed = autoRemoved
	autoRobotNet, _ := validatePostFormFieldNumber("autoRobotNet", r)
	match.Auto.RobotNet = autoRobotNet

	teleopL4, _ := validatePostFormFieldNumber("teleopL4", r)
	match.Teleop.L4 = teleopL4
	teleopL3, _ := validatePostFormFieldNumber("teleopL3", r)
	match.Teleop.L3 = teleopL3
	teleopL2, _ := validatePostFormFieldNumber("teleopL2", r)
	match.Teleop.L2 = teleopL2
	teleopL1, _ := validatePostFormFieldNumber("teleopL1", r)
	match.Teleop.L1 = teleopL1

	teleopProcessor, _ := validatePostFormFieldNumber("teleopProcessor", r)
	match.Teleop.Processor = teleopProcessor
	teleopRemoved, _ := validatePostFormFieldNumber("teleopRemoved", r)
	match.Teleop.Removed = teleopRemoved
	teleopRobotNet, _ := validatePostFormFieldNumber("teleopRobotNet", r)
	match.Teleop.RobotNet = teleopRobotNet
	teleopHumanNet, _ := validatePostFormFieldNumber("teleopHumanNet", r)
	match.Teleop.HumanNet = teleopHumanNet

	barge, exists := model.BargeMap[r.PostFormValue("barge")]
	if !exists {
		match.Endgame.Barge = model.None;
	} else {
		match.Endgame.Barge = barge;
	}

	file, err = os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	json.NewEncoder(file).Encode(match)

	if err = drivers.ExecuteTemplate(w, "edit-match-form.html.tmpl", model.Page[model.Match]{
		Title: fmt.Sprintf("%s Editing Team %d - %s Match %d", match.Meta.ScouterName, match.Meta.TeamNumber, match.Meta.MatchType, match.Meta.MatchNumber),
		Data:  match,
	}); err != nil {
		log.Println(err)
	}
}

func delete(w http.ResponseWriter, r *http.Request) {
	err := os.Remove(filepath.Join(model.Dir, r.PathValue("hash")))
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
	}
}
