package main

import (
	"encoding/json"
	"math"
	"net/http"
	"sync"

	"github.com/hajimehoshi/oto"
)

type Oscillator struct {
	mu         sync.Mutex
	frequency  float64
	sampleRate int
	phase      float64
}

func NewOscillator(freq float64, sampleRate int) *Oscillator {
	return &Oscillator{
		frequency:  freq,
		sampleRate: sampleRate,
		phase:      0,
	}
}

func (o *Oscillator) SetFrequency(freq float64) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.frequency = freq
}

func (o *Oscillator) NextSample() float64 {
	o.mu.Lock()
	defer o.mu.Unlock()
	sample := math.Sin(o.phase * 2 * math.Pi)
	o.phase += o.frequency / float64(o.sampleRate)
	o.phase = math.Mod(o.phase, 1.0)
	return sample
}

func main() {
	oscillator := NewOscillator(440, 44100)

	ctx, err := oto.NewContext(44100, 1, 1, 4096)
	if err != nil {
		panic(err)
	}
	defer ctx.Close()
	player := ctx.NewPlayer()
	defer player.Close()

	http.HandleFunc("/setFrequency", func(w http.ResponseWriter, r *http.Request) {
		var data map[string]float64
		err := json.NewDecoder(r.Body).Decode(&data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		frequency := data["frequency"]
		oscillator.SetFrequency(frequency)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})

	go func() {
		for {
			buf := make([]byte, 4096)
			for i := 0; i < len(buf); i++ {
				sample := oscillator.NextSample()
				buf[i] = byte((sample + 1) / 2 * 255)
			}
			player.Write(buf)
		}
	}()

	http.ListenAndServe(":8080", nil)
}
