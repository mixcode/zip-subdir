
# Create ZIP of each directory

Usually, zipping files is an easy job for simple shell scripts. However, there are some extreme cases that Linux-based shell scripts do not fit well. For example, Windows filesystems may have extremely long file names that Linux cannot handle. Or, there are circumtances that standard ZIP tools are not available.

This utility is a simple stand-alone tool that zip directories. This tool CANNOT handle individual files but is enough for batch processing existing files.

## Install

```sh
go install github.com/mixcode/zip-subdir@latest
```

## Example

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


