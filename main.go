package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	iconv "github.com/djimenez/iconv-go"
	tty "github.com/mattn/go-tty"
)

const (
	UTF8 = "utf-8"
)

// global configs
var (
	omitDirName           = false
	createZipForEmptyDir  = false
	iterateSubdirectories = false

	quiet     = false
	overwrite = false

	convertTo = UTF8 // filename codepage conversion

	//workdir = "."
	outdir = "."

	//h := &zip.FileHeader{Name: zipname, Method: zip.Deflate, Flags: FLAG_EFS}
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
	//const FLAG_EFS = 0x800 // EFS bit: for UTF-8 filename
	//h := &zip.FileHeader{Name: zipname, Method: zip.Deflate, Flags: FLAG_EFS}
	name := zipname
	if convertTo != UTF8 {
		// Note that it's safe to store non-UTF8 bytes in Go string, because it's internally just a []byte
		name, err = iconv.ConvertString(name, UTF8, convertTo)
		if err != nil {
			return
		}
	}
	h := &zip.FileHeader{Name: name, Method: zip.Deflate}
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
// dirpath: the path to the directory
func makeZip(dirpath string) (err error) {
	basename := filepath.Base(dirpath)
	//ospath := filepath.Join(workdir, dirname)
	zipname := filepath.Join(outdir, basename) + ".zip"

	st, err := os.Stat(zipname)
	if !os.IsNotExist(err) {
		if st.IsDir() {
			return fmt.Errorf("cannot create file %s", zipname)
		}
		if !overwrite {
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
		files, err = os.ReadDir(dirpath)
		if err != nil {
			return
		}
		for _, f := range files {
			var st os.FileInfo
			st, err = os.Stat(filepath.Join(dirpath, f.Name()))
			if err != nil {
				return
			}
			err = AddFileInfoToZip(zw, basename, "", st)
			if err != nil {
				return
			}
		}

	} else {

		// add the whole directory
		err = AddFileToZip(zw, "", dirpath)
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
	arg := flag.Args()
	for _, path := range arg {
		var st fs.FileInfo
		st, err = os.Stat(path)
		if os.IsNotExist(err) {
			return
		}
		if !st.IsDir() {
			return fmt.Errorf("%s is not a directory", path)
		}
		if iterateSubdirectories {
			err = iterateDir(path)
		} else {
			err = makeZip(path)
		}
		if err != nil {
			return
		}
	}
	return nil
}

// find subdirectories in a directory and zip each of them
func iterateDir(path string) (err error) {
	files, err := os.ReadDir(path)
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
			f2, err = os.ReadDir(filepath.Join(path, f.Name()))
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
				if !overwrite {
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
		fo := flag.CommandLine.Output()

		fmt.Fprintf(fo, "Compress each directory to a ZIP file\n")
		fmt.Fprintf(fo, "\n")
		fmt.Fprintf(fo, "Usage: %s [flags] directory [directory...]\n", os.Args[0])
		fmt.Fprintf(fo, "\n")
		fmt.Fprintf(fo, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(fo, "\n")
	}

	flag.BoolVar(&omitDirName, "c", omitDirName, "contents mode; the directory name is omitted in new zip files")
	flag.BoolVar(&iterateSubdirectories, "s", iterateSubdirectories, "scan subdirectories of the directories and zip each of them")
	flag.BoolVar(&quiet, "q", quiet, "suppress progress outputs")
	flag.BoolVar(&overwrite, "o", overwrite, "Force; overwrite everything without asking")
	flag.BoolVar(&createZipForEmptyDir, "e", createZipForEmptyDir, "create ZIP even for empty subdirectories")
	flag.StringVar(&outdir, "d", outdir, "output directory to put created ZIP files")

	flag.StringVar(&convertTo, "t", convertTo, "codepage of filenames in created zip. WARNING: use only if you know exactly what you are doing!")

	flag.Parse()

	/*
		workdir = flag.Arg(0)
		if workdir == "" {
			workdir = "."
		}
	*/

	if outdir == "" {
		outdir = "."
	}

	err := run()

	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
