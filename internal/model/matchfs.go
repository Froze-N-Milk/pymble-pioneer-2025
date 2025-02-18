package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sync"
)

var matchMap = sync.Map{}
var matchMapMutex = sync.RWMutex{}
var Dir string

func Init() {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	Dir = filepath.Join(home, "pymble-pioneer")
	if err = os.Mkdir(Dir, os.ModePerm); err != nil && !errors.Is(err, fs.ErrExist) {
		log.Fatal(err)
	}

	CleanFS()
}

func CleanFS() {
	if err := filepath.WalkDir(Dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type().IsDir() {
			return filepath.SkipDir
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}

		decoder := json.NewDecoder(file)
		match := Match{}
		err = decoder.Decode(&match)
		if err != nil {
			err = os.Remove(Dir)
			if err != nil {
				return err
			}
		}

		hash := match.Meta.Hash()
		filename := fmt.Sprintf("%x.json", hash)
		if d.Name() != filename {
			os.Rename(path, filepath.Join(filepath.Dir(path), filename))
		}

		return nil
	}); err != nil {
		log.Fatal(err)
	}
}
