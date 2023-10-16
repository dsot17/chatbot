package utils

func Map[T, U any](elements []T, f func(T) U) []U {
    mappedElements := make([]U, len(elements))

    for i := range elements {
        mappedElements[i] = f(elements[i])
    }

    return mappedElements
}
