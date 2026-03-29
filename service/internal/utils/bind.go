package utils

func Bind1[T any, U any](dst *U, fn func(T) (U, error), arg T) func() error {
	return func() error {
		res, err := fn(arg)
		if err != nil {
			return err
		}
		*dst = res
		return nil
	}
}

func Bind0[U any](dst *U, fn func() (U, error)) func() error {
	return func() error {
		res, err := fn()
		if err != nil {
			return err
		}
		*dst = res
		return nil
	}
}
