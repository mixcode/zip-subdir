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

	tty "github.com/mattn/go-tty"
)

const (
	UTF8 = "utf-8"
)

// global configs
var (
	keepDirName           = false
	createZipForEmptyDir  = false
	iterateSubdirectories = false

	quiet     = false
	overwrite = false

	convertTo = UTF8 // filename codepage conversion

	destDir = "."
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
	var h *zip.FileHeader
	name := zipname
	if useIconv() && convertTo != UTF8 {
		// convert the filename to a different charset
		var nonUTF8 bool
		name, nonUTF8, err = convertCharsetFrom(convertTo, name)
		//name, err = iconv.ConvertString(name, UTF8, convertTo) // Note that it's safe to store non-UTF8 bytes in Go string, because it's internally just a []byte
		if err != nil {
			return
		}
		h = &zip.FileHeader{Name: name, Method: zip.Deflate, NonUTF8: nonUTF8}
	} else {
		h = &zip.FileHeader{Name: name, Method: zip.Deflate, NonUTF8: false}
	}
	h.Modified = time.Now()
	fz, err := z.CreateHeader(h)
	if err != nil {
		return
	}
	if !quiet {
		fmt.Printf("\t%s\n", zipname)
	}
	_, err = io.Copy(fz, f)
	return
}

// make a zip for a subdirectory
// dirpath: the path to the directory
func makeZip(dirpath string) (err error) {

	dirpath = strings.TrimRight(dirpath, "/\\") // remove trailing slashes

	basename := filepath.Base(dirpath)
	zipname := filepath.Join(destDir, basename) + ".zip"

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
		fmt.Printf("%s\n", zipname)
	}
	zw := zip.NewWriter(fi)
	defer zw.Close()

	if !keepDirName {

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
			err = AddFileInfoToZip(zw, dirpath, "", st)
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

// find subdirectories in a directory and zip each of them
func iterateDir(path string) (err error) {
	files, err := os.ReadDir(path)
	if err != nil {
		return
	}

	for _, f := range files {
		if !f.IsDir() {
			continue
		}
		if !createZipForEmptyDir {
			// check the subdir is empty or not
			var d []os.DirEntry
			d, err = os.ReadDir(filepath.Join(path, f.Name()))
			if err != nil {
				return
			}
			if len(d) == 0 {
				// ignore the empty directory
				continue
			}
		}
		err = makeZip(filepath.Join(path, f.Name()))
		if err != nil {
			return
		}
	}

	return
}

// actual main
func run() (err error) {
	arg := flag.Args()
	if len(arg) == 0 {
		return fmt.Errorf("no directory given (use --help for help)")
	}

	if overwrite {
		// create the destionation directory if not exists
		_, e := os.Stat(destDir)
		if os.IsNotExist(e) {
			err = os.MkdirAll(destDir, fs.ModePerm)
			if err != nil {
				return
			}
		}
	}

	// expand wildcard pattern and find directories
	files := make([]string, 0)
	for _, a := range arg {
		var l []string
		l, err = filepath.Glob(a)
		if err != nil {
			return
		}
		if len(l) == 0 {
			// expansion failed
			files = append(files, a)
			continue
		}
		if len(l) == 1 {
			// only one entry found; do not filter directories
			files = append(files, l[0])
			continue
		}
		// multiple entries: select directories
		for _, f := range l {
			st, _ := os.Stat(f)
			if st != nil && st.IsDir() {
				files = append(files, f)
			}
		}
	}

	for _, path := range files {
		var st fs.FileInfo
		st, err = os.Stat(path)
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "\"%s\" not found\n", path)
			continue
		}
		if st == nil || !st.IsDir() {
			fmt.Fprintf(os.Stderr, "\"%s\" is not a directory\n", path)
			continue
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

func main() {

	flag.Usage = func() {
		fo := flag.CommandLine.Output()

		fmt.Fprintf(fo, "Compress each directory to a zip file.\n")
		fmt.Fprintf(fo, "The created zip files will have the same name with the directory.\n")
		fmt.Fprintf(fo, "\n")
		fmt.Fprintf(fo, "Usage: %s [flags] directory [directory...]\n", os.Args[0])
		fmt.Fprintf(fo, "\n")
		fmt.Fprintf(fo, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(fo, "\n")
	}

	flag.BoolVar(&keepDirName, "k", keepDirName, "keep the directory name included in the filenames in the zip")
	flag.BoolVar(&iterateSubdirectories, "s", iterateSubdirectories, "scan subdirectories of the directories and zip each of them")
	flag.BoolVar(&quiet, "q", quiet, "quiet; suppress messages")
	flag.BoolVar(&overwrite, "o", overwrite, "overwrite file without asking. also creates the destination directory if not exists.")
	flag.BoolVar(&createZipForEmptyDir, "e", createZipForEmptyDir, "with -s, create ZIP even for empty subdirectories")
	flag.StringVar(&destDir, "d", destDir, "destination directory to put created zip files")

	if useIconv() {
		flag.StringVar(&convertTo, "t", convertTo, "iconv charset name of filenames in created zip. WARNING: use only if you know exactly what you are doing!")
	}

	flag.Parse()

	if destDir == "" {
		destDir = "."
	}

	err := run()

	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
