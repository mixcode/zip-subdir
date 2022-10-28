
# Create ZIPs of each subdirectory

Usually, zipping files is an easy job for simple shell scripts. However, there are some extreme cases that Linux-based shell scripts do not fit well. For example, Windows filesystems may have extremely long file names that Linux cannot handle. Or, there are circumtances that standard ZIP tools are not available.

This utility is a stand-alone tool that scans subdirectories and compress each directory to a separate, independent ZIP file.


## Install

```
go install github.com/mixcode/zip-subdir
```

## Usage

* Zip each subdirectory of the current directory. Each ZIP file's top directory contains the subdirectory itself.
```
zip-subdir
```

* Zip subdirectories of the current directory. Each ZIP file's top directory has contents of the subdirectory.
```
zip-subdir -c
```

* Create ZIPs to another directory, silently overwriting all existing files, including the empty directories.
```
zip-subdir -q -e -o SOME_OUTPUT_DIRECTORY -f SOME_SOURCE_DIRECTORY
```

* Show flag list
```
zip-subdir -h
```


