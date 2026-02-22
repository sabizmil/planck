package ui

import "time"

// SpinnerPreset defines a named spinner animation.
type SpinnerPreset struct {
	Name     string
	Frames   []string
	Interval time.Duration
}

// reverseMirror expands frames into a forward+backward sequence, omitting the
// first and last frames on the way back to avoid a stutter at the endpoints.
// e.g. [a, b, c] → [a, b, c, b]
func reverseMirror(frames []string) []string {
	if len(frames) <= 2 {
		return frames
	}
	result := make([]string, 0, len(frames)*2-2)
	result = append(result, frames...)
	for i := len(frames) - 2; i >= 1; i-- {
		result = append(result, frames[i])
	}
	return result
}

// spinnerPresets is the internal registry of all available presets.
var spinnerPresets []SpinnerPreset

func init() {
	type rawPreset struct {
		name     string
		frames   []string
		interval time.Duration
		reverse  bool
	}

	raw := []rawPreset{
		// Default: Claude-style asterisk breathing
		{name: "claude", frames: []string{"·", "✢", "✳", "✶", "✻", "✽"}, interval: 120 * time.Millisecond, reverse: true},

		// Dot pulse (the old hardcoded default)
		{name: "dot-pulse", frames: []string{"·", "•", "●", "•"}, interval: 250 * time.Millisecond},

		// Classic dots
		{name: "dots", frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}, interval: 80 * time.Millisecond},

		// Line
		{name: "line", frames: []string{"-", "\\", "|", "/"}, interval: 130 * time.Millisecond},

		// Star
		{name: "star", frames: []string{"✶", "✸", "✹", "✺", "✹", "✷"}, interval: 100 * time.Millisecond},

		// Flip
		{name: "flip", frames: []string{"_", "_", "_", "-", "`", "`", "'", "´", "-", "_", "_", "_"}, interval: 70 * time.Millisecond},

		// Bounce
		{name: "bounce", frames: []string{"⠁", "⠂", "⠄", "⠂"}, interval: 120 * time.Millisecond},

		// Box bounce
		{name: "box-bounce", frames: []string{"▖", "▘", "▝", "▗"}, interval: 120 * time.Millisecond},

		// Arc
		{name: "arc", frames: []string{"◜", "◠", "◝", "◞", "◡", "◟"}, interval: 100 * time.Millisecond},

		// Circle quarters
		{name: "circle", frames: []string{"◴", "◷", "◶", "◵"}, interval: 120 * time.Millisecond},

		// Circle halves
		{name: "circle-half", frames: []string{"◐", "◓", "◑", "◒"}, interval: 120 * time.Millisecond},

		// Square corners
		{name: "square-corners", frames: []string{"◰", "◳", "◲", "◱"}, interval: 120 * time.Millisecond},

		// Triangle
		{name: "triangle", frames: []string{"◢", "◣", "◤", "◥"}, interval: 120 * time.Millisecond},

		// Binary
		{name: "binary", frames: []string{"010010", "001100", "100101", "111010", "001011", "010101"}, interval: 100 * time.Millisecond},

		// Toggle
		{name: "toggle", frames: []string{"⊶", "⊷"}, interval: 250 * time.Millisecond},

		// Arrow
		{name: "arrow", frames: []string{"←", "↖", "↑", "↗", "→", "↘", "↓", "↙"}, interval: 100 * time.Millisecond},

		// Balloon
		{name: "balloon", frames: []string{".", "o", "O", "@", "*", " "}, interval: 140 * time.Millisecond},

		// Noise
		{name: "noise", frames: []string{"▓", "▒", "░"}, interval: 100 * time.Millisecond, reverse: true},

		// Grow horizontal
		{name: "grow-h", frames: []string{"▏", "▎", "▍", "▌", "▋", "▊", "▉", "█"}, interval: 100 * time.Millisecond, reverse: true},

		// Grow vertical
		{name: "grow-v", frames: []string{"▁", "▃", "▄", "▅", "▆", "▇", "█"}, interval: 100 * time.Millisecond, reverse: true},

		// Layer
		{name: "layer", frames: []string{"-", "=", "≡"}, interval: 150 * time.Millisecond, reverse: true},

		// Moon
		{name: "moon", frames: []string{"🌑", "🌒", "🌓", "🌔", "🌕", "🌖", "🌗", "🌘"}, interval: 100 * time.Millisecond},

		// Hearts
		{name: "hearts", frames: []string{"💛", "💙", "💜", "💚", "❤️"}, interval: 120 * time.Millisecond},

		// Clock
		{name: "clock", frames: []string{"🕐", "🕑", "🕒", "🕓", "🕔", "🕕", "🕖", "🕗", "🕘", "🕙", "🕚", "🕛"}, interval: 100 * time.Millisecond},

		// Point
		{name: "point", frames: []string{"∙∙∙", "●∙∙", "∙●∙", "∙∙●", "∙∙∙"}, interval: 125 * time.Millisecond},

		// Meter
		{name: "meter", frames: []string{"▱▱▱", "▰▱▱", "▰▰▱", "▰▰▰", "▰▰▱", "▰▱▱"}, interval: 150 * time.Millisecond},

		// Breathing dot
		{name: "breathe", frames: []string{"·", "•", "●", "•", "·"}, interval: 160 * time.Millisecond},
	}

	spinnerPresets = make([]SpinnerPreset, 0, len(raw))
	for _, r := range raw {
		frames := r.frames
		if r.reverse {
			frames = reverseMirror(frames)
		}
		spinnerPresets = append(spinnerPresets, SpinnerPreset{
			Name:     r.name,
			Frames:   frames,
			Interval: r.interval,
		})
	}
}

// SpinnerPresets returns all available spinner presets.
func SpinnerPresets() []SpinnerPreset {
	return spinnerPresets
}

// SpinnerPresetByName looks up a preset by name, falling back to the default.
func SpinnerPresetByName(name string) SpinnerPreset {
	for _, p := range spinnerPresets {
		if p.Name == name {
			return p
		}
	}
	return spinnerPresets[0] // "claude" is always first
}

// DefaultSpinnerPreset returns the name of the default spinner preset.
func DefaultSpinnerPreset() string {
	return "claude"
}
