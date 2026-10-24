// Package textedit is the model behind a text field that the platform keeps outside the framework (the
// hidden DOM field of the web, SPEC §23.5, and the Editable of the Android input method, SPEC §24.6), as
// plain Go: the editing state in UTF-16 units, the composition, and the editor that turns each change
// of the platform's field into the ui.TextInputEvent that makes the framework's field the same. It has
// no operating system code, so the web and Android platforms share it.
package textedit
