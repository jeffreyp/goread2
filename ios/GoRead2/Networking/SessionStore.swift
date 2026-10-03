import Foundation
import Security

/// The session credential from the mobile auth handoff: the cookie name the
/// backend reads and the session ID it carries.
struct SessionCredential: Codable, Equatable {
    let cookieName: String
    let token: String
}

/// Keychain storage for the session credential, one generic-password item
/// per backend host so Debug and Release builds keep separate sessions.
///
/// On iOS the item is readable after the first unlock following a reboot
/// and never leaves the device, neither through backups nor iCloud Keychain.
/// On macOS it lives in the login keychain, guarded by an access list that
/// trusts this app's signature. The data protection keychain would need a
/// provisioning-profile application identifier that the Mac builds do not
/// carry.
struct SessionStore {
    private static let service = "org.jeffreypratt.goread2.session"

    let account: String

    init(baseURL: URL) {
        account = baseURL.host ?? baseURL.absoluteString
    }

    func load() -> SessionCredential? {
        var query = baseQuery
        query[kSecReturnData as String] = true
        query[kSecMatchLimit as String] = kSecMatchLimitOne

        var result: CFTypeRef?
        guard SecItemCopyMatching(query as CFDictionary, &result) == errSecSuccess,
              let data = result as? Data else { return nil }
        return try? JSONDecoder().decode(SessionCredential.self, from: data)
    }

    func save(_ credential: SessionCredential) throws {
        let data = try JSONEncoder().encode(credential)
        let attributes: [String: Any] = [
            kSecValueData as String: data,
            kSecAttrAccessible as String: kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly,
        ]

        var status = SecItemUpdate(baseQuery as CFDictionary, attributes as CFDictionary)
        if status == errSecItemNotFound {
            let item = baseQuery.merging(attributes) { _, new in new }
            status = SecItemAdd(item as CFDictionary, nil)
        }
        guard status == errSecSuccess else { throw KeychainError(status: status) }
    }

    func clear() {
        SecItemDelete(baseQuery as CFDictionary)
    }

    private var baseQuery: [String: Any] {
        [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: Self.service,
            kSecAttrAccount as String: account,
        ]
    }
}

struct KeychainError: LocalizedError {
    let status: OSStatus

    var errorDescription: String? {
        let message = SecCopyErrorMessageString(status, nil) as String? ?? "status \(status)"
        return "The session could not be saved to the Keychain: \(message)"
    }
}
