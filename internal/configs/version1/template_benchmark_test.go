package version1

import (
	"bytes"
	"html/template"
	"os"
	"runtime/pprof"
	"testing"
)

func BenchmarkExecuteMainTemplateForNGINXPlus(b *testing.B) {
	tmpl, err := template.New("nginx-plus.tmpl").Funcs(helperFunctions).ParseFiles("nginx-plus.tmpl")
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	for range b.N {
		buf := &bytes.Buffer{}
		err := tmpl.Execute(buf, mainCfg)
		if err != nil {
			b.Fatal(err)
		}
		createFileAndWrite("configfile", buf.Bytes())
	}
}

func BenchmarkExecuteMainTemplateForNGINXPlusDirectWrite(b *testing.B) {
	tmpl, err := template.New("nginx-plus.tmpl").Funcs(helperFunctions).ParseFiles("nginx-plus.tmpl")
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	for range b.N {
		f, err := os.Create("configfile")
		if err != nil {
			b.Fatal(err)
		}
		err = tmpl.Execute(f, mainCfg)
		if err != nil {
			b.Fatal(err)
		}
		f.Close()
	}
}

func BenchmarkExecuteMainTemplateForNGINXPlusPProf(b *testing.B) {
	tmpl, err := template.New("nginx-plus.tmpl").Funcs(helperFunctions).ParseFiles("nginx-plus.tmpl")
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	// Create file for saving CPU profile data
	f, err := os.Create("cpuprofile")
	if err != nil {
		b.Fatal(err)
	}

	// Start profiling and run benchmark
	err = pprof.StartCPUProfile(f)
	if err != nil {
		b.Fatal(err)
	}

	for range b.N {
		buf := &bytes.Buffer{}
		err := tmpl.Execute(buf, mainCfg)
		if err != nil {
			b.Fatal(err)
		}
		createFileAndWrite("configfile", buf.Bytes())
	}

	pprof.StopCPUProfile()
}

func createFileAndWrite(name string, b []byte) error {
	w, err := os.Create(name)
	if err != nil {
		return err
	}

	defer func() {
		if tempErr := w.Close(); tempErr != nil {
			err = tempErr
		}
	}()

	_, err = w.Write(b)
	if err != nil {
		return err
	}

	return err
}
