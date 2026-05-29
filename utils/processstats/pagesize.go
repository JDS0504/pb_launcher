package processstats

import "os"

// pageSize devuelve el tamaño de página de memoria del sistema operativo en bytes.
// Es una función auxiliar para convertir RSS de páginas a bytes de forma portable.
func pageSize() int {
	return os.Getpagesize()
}
