
# Create ZIPs of directories

Usually, zipping files is an easy job for simple shell scripts. However, there are some extreme cases that Linux-based shell scripts do not fit well. For example, Windows filesystems may have extremely long file names that Linux cannot handle. Or, there are circumtances that standard ZIP tools are not available.

This utility is a simple stand-alone tool that zip directories. This tool CANNOT handle individual files but is enough for batch processing the existing files.

## Install

```sh
go install github.com/mixcode/zip-subdir@latest
```

## Examples

Zip directories `a`, `b`, `c` into `a.zip`, `b.zip`, `c.zip`. The zip files are created in the current directory.
Also note that the trailing slashes are ignored.

```sh
zip-subdir a b/ c
```

---

Zip each subdirectory of the current directory, one by one, then put them into `../outdir`.
Option `-o` also overwrites existing files.

```sh
zip-subdir -s -d=../outdir -o .
```

---

Show the help.

```sh
zip-subdir -help
```

## Optional filename charset conversion

When the command line is built with `-tags iconv` build tag, then the command will have an optional `-t` flag that converts filenames to the desired charset.

libiconv is required.

```sh
go build -tags iconv
```

### Example

Create a zip file which the filenames are stored in Japanese Shift-JIS character encoding. Flag `-k` keeps the directory name ('日本語ファイル名') for the filenames in zip.

```sh
zip-subdir -t=SHIFT-JIS -k 日本語ファイル名/
```
