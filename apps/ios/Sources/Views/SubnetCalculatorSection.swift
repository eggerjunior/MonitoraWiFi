import SwiftUI

/// Calculadora de sub-rede IPv4 (Fase 5): cálculo matemático puro, sem
/// chamada de rede/agente — paridade com
/// apps/web/src/components/SubnetCalculator.tsx.
struct SubnetCalculatorSection: View {
    @Environment(\.colorScheme) private var colorScheme
    @State private var input = "192.168.1.0/24"

    var body: some View {
        Section("Calculadora de sub-rede") {
            TextField("Endereço/prefixo (CIDR)", text: $input)
                .textInputAutocapitalization(.never)
                .autocorrectionDisabled()
                .keyboardType(.numbersAndPunctuation)

            switch SubnetCalculator.parse(input) {
            case .failure(let message):
                Text(message)
                    .foregroundStyle(Color.egger(.critical, scheme: colorScheme))
            case .success(let info):
                LabeledContent("Máscara", value: info.mask)
                LabeledContent("Endereço de rede", value: info.network)
                LabeledContent("Broadcast", value: info.broadcast)
                LabeledContent("Primeiro host", value: info.firstHost)
                LabeledContent("Último host", value: info.lastHost)
                LabeledContent("Hosts utilizáveis", value: String(info.usableHosts))
            }
        }
    }
}

enum SubnetCalculator {
    struct Info {
        let mask: String
        let network: String
        let broadcast: String
        let firstHost: String
        let lastHost: String
        let usableHosts: Int
    }

    enum ParseResult {
        case success(Info)
        case failure(String)
    }

    static func parse(_ input: String) -> ParseResult {
        let parts = input.trimmingCharacters(in: .whitespaces).split(separator: "/")
        guard parts.count == 2,
              let prefix = Int(parts[1]), (0...32).contains(prefix)
        else {
            return .failure("Formato esperado: A.B.C.D/prefixo (ex.: 192.168.1.0/24)")
        }

        let octetStrings = parts[0].split(separator: ".")
        guard octetStrings.count == 4 else {
            return .failure("Formato esperado: A.B.C.D/prefixo (ex.: 192.168.1.0/24)")
        }
        var octets: [UInt32] = []
        for s in octetStrings {
            guard let value = UInt32(s), value <= 255 else {
                return .failure("Endereço ou prefixo fora do intervalo válido")
            }
            octets.append(value)
        }

        let ipInt = (octets[0] << 24) | (octets[1] << 16) | (octets[2] << 8) | octets[3]
        let maskInt: UInt32 = prefix == 0 ? 0 : (0xFFFFFFFF as UInt32) << (32 - prefix)
        let networkInt = ipInt & maskInt
        let broadcastInt = networkInt | ~maskInt

        let totalHosts = prefix >= 32 ? 1 : (1 << (32 - prefix))
        let usableHosts = prefix >= 31 ? 0 : totalHosts - 2

        return .success(Info(
            mask: intToIP(maskInt),
            network: intToIP(networkInt),
            broadcast: intToIP(broadcastInt),
            firstHost: usableHosts > 0 ? intToIP(networkInt + 1) : intToIP(networkInt),
            lastHost: usableHosts > 0 ? intToIP(broadcastInt - 1) : intToIP(broadcastInt),
            usableHosts: usableHosts
        ))
    }

    private static func intToIP(_ value: UInt32) -> String {
        [(value >> 24) & 255, (value >> 16) & 255, (value >> 8) & 255, value & 255]
            .map(String.init)
            .joined(separator: ".")
    }
}

#Preview {
    List {
        SubnetCalculatorSection()
    }
}
