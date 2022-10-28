package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	tty "github.com/mattn/go-tty"
)

// global configs
var (
	omitDirName          = false
	quiet                = false
	force                = false
	createZipForEmptyDir = false

	workdir = "."
	outdir  = "."
)

// show Yes/No prompt
func promptYN(msg string, defaultYes bool) bool {
	tt, err := tty.Open()
	if err != nil {
		return defaultYes
	}
	defer tt.Close()

	fmt.Print(msg)
	r, err := tt.ReadRune()
	fmt.Print("\n")
	if err == nil {
		s := strings.ToLower(string(r))
		if s == "y" {
			return true
		} else if s == "n" {
			return false
		}
	}
	return defaultYes
}

func AddFileToZip(z *zip.Writer, zipprefix string, pathToFile string) (err error) {
	st, err := os.Stat(pathToFile)
	if err != nil {
		return
	}
	pathPrefix, _ := filepath.Split(pathToFile)
	return AddFileInfoToZip(z, pathPrefix, zipprefix, st)
}

func AddFileInfoToZip(z *zip.Writer, pathprefix string, zipprefix string, fi os.FileInfo) (err error) {

	var f *os.File
	pathname := filepath.Join(pathprefix, fi.Name())
	zipname := filepath.Join(zipprefix, fi.Name())
	f, err = os.Open(pathname)
	if err != nil {
		return
	}
	defer func() {
		e := f.Close()
		if err == nil {
			err = e
		}
	}()

	if fi.IsDir() {
		// process a directory
		var subfi []os.FileInfo
		subfi, err = f.Readdir(0)
		if err != nil {
			return
		}
		for _, sf := range subfi {
			err = AddFileInfoToZip(z, pathname, zipname, sf)
			if err != nil {
				return
			}
		}
		return
	}

	// process a single file
	const FLAG_EFS = 0x800 // EFS bit: for UTF-8 filename
	h := &zip.FileHeader{Name: zipname, Method: zip.Deflate, Flags: FLAG_EFS}
	//h.SetModTime(time.Now())	// deprecated
	h.Modified = time.Now()
	out, err := z.CreateHeader(h)
	if err != nil {
		return
	}
	if !quiet {
		fmt.Printf("%s\n", pathname)
	}
	_, err = io.Copy(out, f)
	return
}

// make a zip for a subdirectory
func makeZip(dirbasename string) (err error) {
	ospath := filepath.Join(workdir, dirbasename)
	zipname := filepath.Join(outdir, dirbasename) + ".zip"

	st, err := os.Stat(zipname)
	if !os.IsNotExist(err) {
		if st.IsDir() {
			return fmt.Errorf("cannot create file %s", zipname)
		}
		if !force {
			fmt.Printf("The output file '%s' already exists.", zipname)
			yes := promptYN(" Overwrite? (y/N)", false)
			if !yes {
				// ignore this file
				return nil
			}
		}
	}

	fi, err := os.Create(zipname)
	if err != nil {
		return
	}
	defer fi.Close()
	if !quiet {
		fmt.Printf("Creating %s\n", zipname)
	}
	zw := zip.NewWriter(fi)
	defer zw.Close()

	if omitDirName {

		// add each contents
		var files []os.DirEntry
		files, err = os.ReadDir(ospath)
		if err != nil {
			return
		}
		for _, f := range files {
			var st os.FileInfo
			st, err = os.Stat(filepath.Join(ospath, f.Name()))
			if err != nil {
				return
			}
			err = AddFileInfoToZip(zw, dirbasename, "", st)
			if err != nil {
				return
			}
		}

	} else {

		// add the whole directory
		err = AddFileToZip(zw, "", ospath)
		if err != nil {
			return
		}

	}

	if !quiet {
		// write a newline
		fmt.Println()
	}
	return
}

// actual main
func run() (err error) {
	files, err := os.ReadDir(workdir)
	if err != nil {
		return
	}

	outdirOk := false

	for _, f := range files {
		if !f.IsDir() {
			continue
		}
		// Check contents of the dirctory
		if !createZipForEmptyDir {
			// check the contents of the subdir
			var f2 []os.DirEntry
			f2, err = os.ReadDir(filepath.Join(workdir, f.Name()))
			if err != nil {
				return
			}
			if len(f2) == 0 {
				continue
			}
		}
		if !outdirOk {
			// check the output directory
			var st os.FileInfo
			st, err = os.Stat(outdir)
			if os.IsNotExist(err) {
				// create the output directory
				if !force {
					yes := promptYN("The output directory does not exists. Create? (y/N)", false)
					if !yes {
						// stop by user interevention
						return fmt.Errorf("stop")
					}
				}
				err = os.MkdirAll(outdir, 0644)
				if err != nil {
					return
				}
			} else {
				if !st.IsDir() {
					err = fmt.Errorf("output path is not a directory")
					return
				}
			}
			outdirOk = true
		}
		err = makeZip(f.Name())
		if err != nil {
			return
		}
	}

	return
}

func main() {

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Compress each subdirectory to different ZIP files\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [flags] [directory]\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.BoolVar(&omitDirName, "c", omitDirName, "contents mode; the top subdirectory name is omitted in new zip files")
	flag.BoolVar(&quiet, "q", quiet, "suppress progress outputs")
	flag.BoolVar(&force, "f", force, "Force; overwrite everything without asking")
	flag.BoolVar(&createZipForEmptyDir, "e", createZipForEmptyDir, "create ZIP even for empty subdirectories")
	flag.StringVar(&outdir, "o", outdir, "output directory to put created ZIP files")
	flag.Parse()

	workdir = flag.Arg(0)
	if workdir == "" {
		workdir = "."
	}
	if outdir == "" {
		outdir = "."
	}

	err := run()

	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
