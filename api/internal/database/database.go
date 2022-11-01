package database


type CubeEntry struct {
	StorageAccount   string
	Container        string
	Country          string
	Field            string
	FieldRestricted  bool
	FilenameOnUpload string
}

/** Database interface
 *
 * Interface for communication with a database / index. Any data source that
 * implements this interface could be used as a backend for the Catalogue API.
 */
type Adapter interface {
	/** Get Cubes 
	 *
	 * Fetches and returns all cubes where CubeEntry.Field is in fields.
	 */
	GetCubes(fields []string) ([]CubeEntry, error)
}
