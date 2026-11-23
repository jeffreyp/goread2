import Foundation
import Network

/// Whether the device has a network path, from `NWPathMonitor`. The screens
/// use it to disable actions that need the server while there is none, and
/// to reload once a connection returns.
///
/// A satisfied path does not prove the server is reachable, so the view
/// models also treat a request that fails with `NetworkError.noConnection`
/// as being offline until a later request succeeds.
@MainActor
final class ConnectivityMonitor: ObservableObject {
    static let shared = ConnectivityMonitor()

    @Published private(set) var isOffline = false

    private let monitor = NWPathMonitor()

    private init() {
        monitor.pathUpdateHandler = { [weak self] path in
            let isOffline = path.status != .satisfied
            Task { @MainActor [weak self] in
                guard let self, self.isOffline != isOffline else { return }
                self.isOffline = isOffline
            }
        }
        monitor.start(queue: DispatchQueue(label: "org.jeffreypratt.goread2.connectivity", qos: .utility))
    }
}
