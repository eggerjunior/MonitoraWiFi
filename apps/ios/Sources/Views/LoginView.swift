import SwiftUI

struct LoginView: View {
    @Environment(SessionStore.self) private var session
    @Environment(\.colorScheme) private var colorScheme

    @State private var email = ""
    @State private var password = ""
    @State private var isSubmitting = false

    var body: some View {
        VStack(spacing: 24) {
            VStack(spacing: 4) {
                Text("Egger Network Intelligence")
                    .font(.title2.bold())
                Text("Entre com seu e-mail e senha.")
                    .font(.subheadline)
                    .foregroundStyle(Color.egger(.textSecondary, scheme: colorScheme))
            }
            .multilineTextAlignment(.center)

            VStack(spacing: 12) {
                TextField("E-mail", text: $email)
                    .textContentType(.username)
                    .keyboardType(.emailAddress)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
                    .padding(12)
                    .background(.thinMaterial, in: RoundedRectangle(cornerRadius: 10))

                SecureField("Senha", text: $password)
                    .textContentType(.password)
                    .padding(12)
                    .background(.thinMaterial, in: RoundedRectangle(cornerRadius: 10))
            }

            if let error = session.lastError {
                Text(error)
                    .font(.footnote)
                    .foregroundStyle(Color.egger(.critical, scheme: colorScheme))
                    .accessibilityAddTraits(.isStaticText)
            }

            Button {
                Task {
                    isSubmitting = true
                    await session.login(email: email, password: password)
                    isSubmitting = false
                }
            } label: {
                if isSubmitting {
                    ProgressView().tint(.white)
                } else {
                    Text("Entrar").frame(maxWidth: .infinity)
                }
            }
            .buttonStyle(.borderedProminent)
            .tint(Color.egger(.accent, scheme: colorScheme))
            .disabled(email.isEmpty || password.count < 8 || isSubmitting)
        }
        .padding(24)
        .frame(maxWidth: 400)
    }
}

#Preview {
    LoginView()
        .environment(SessionStore())
}
