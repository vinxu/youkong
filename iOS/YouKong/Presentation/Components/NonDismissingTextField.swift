import SwiftUI
import UIKit

/// UITextField wrapper that keeps the keyboard open after pressing Send/Return.
/// SwiftUI's TextField `.onSubmit` forcefully resigns first responder, causing
/// keyboard dismiss→reappear flicker. This wrapper returns `false` from
/// `textFieldShouldReturn` to prevent that.
///
/// IMPORTANT: `onSend` receives the trimmed text but does NOT auto-clear.
/// The caller is responsible for clearing `text` when appropriate.
struct NonDismissingTextField: UIViewRepresentable {
    @Binding var text: String
    var placeholder: String
    var placeholderColor: UIColor
    var textColor: UIColor
    var font: UIFont
    var isFocused: Binding<Bool>
    var onSend: (String) -> Void

    func makeCoordinator() -> Coordinator {
        Coordinator(self)
    }

    func makeUIView(context: Context) -> UITextField {
        let tf = UITextField()
        tf.delegate = context.coordinator
        tf.returnKeyType = .send
        tf.font = font
        tf.textColor = textColor
        tf.attributedPlaceholder = NSAttributedString(
            string: placeholder,
            attributes: [.foregroundColor: placeholderColor]
        )
        tf.autocorrectionType = .default
        tf.backgroundColor = .clear
        // Respect intrinsic height so it doesn't stretch in flexible layouts
        tf.setContentHuggingPriority(.required, for: .vertical)
        tf.setContentCompressionResistancePriority(.required, for: .vertical)
        tf.addTarget(
            context.coordinator,
            action: #selector(Coordinator.textChanged(_:)),
            for: .editingChanged
        )
        return tf
    }

    func updateUIView(_ tf: UITextField, context: Context) {
        if tf.text != text {
            tf.text = text
        }
        let shouldFocus = isFocused.wrappedValue
        if shouldFocus && !tf.isFirstResponder {
            DispatchQueue.main.async { tf.becomeFirstResponder() }
        } else if !shouldFocus && tf.isFirstResponder {
            DispatchQueue.main.async { tf.resignFirstResponder() }
        }
    }

    class Coordinator: NSObject, UITextFieldDelegate {
        var parent: NonDismissingTextField

        init(_ parent: NonDismissingTextField) {
            self.parent = parent
        }

        @objc func textChanged(_ tf: UITextField) {
            parent.text = tf.text ?? ""
        }

        func textFieldShouldReturn(_ tf: UITextField) -> Bool {
            let trimmed = (tf.text ?? "").trimmingCharacters(in: .whitespaces)
            if !trimmed.isEmpty {
                parent.onSend(trimmed)
            }
            // return false → keyboard stays open
            return false
        }

        func textFieldDidBeginEditing(_ tf: UITextField) {
            parent.isFocused.wrappedValue = true
        }

        func textFieldDidEndEditing(_ tf: UITextField) {
            parent.isFocused.wrappedValue = false
        }
    }
}
