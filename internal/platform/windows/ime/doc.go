// Package ime is the text input of the Windows backend as plain Go: the composition of an input
// method (IMM32, SPEC §25.7) turned into the ui.TextInputEvents of the framework by the model of
// internal/platform/textedit, the characters of the keyboard that no key made, and the places of
// the composition and candidate windows. It talks to Windows through the small interface IMM, which
// imm_windows.go implements with imm32.dll and the tests with a fake, so the model runs on any host.
//
// Flutter: shell/platform/windows/text_input_manager.cc TextInputManager
package ime
