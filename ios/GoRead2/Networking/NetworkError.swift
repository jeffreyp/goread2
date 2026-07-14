import Foundation

enum NetworkError: Error, LocalizedError {
    /// 401: not signed in or the session expired.
    case unauthorized
    /// 404: the resource does not exist.
    case notFound
    /// 402: subscription gating (feed limit reached or trial expired).
    case paymentRequired(message: String)
    /// Any other non-2xx response.
    case serverError(statusCode: Int, message: String)
    /// The request never reached the server.
    case noConnection
    /// The response body did not match the expected shape.
    case decodingError(underlying: Error)

    var errorDescription: String? {
        switch self {
        case .unauthorized:
            return "You must be signed in to access this resource."
        case .notFound:
            return "The requested resource could not be found."
        case .paymentRequired(let message):
            return message
        case .serverError(_, let message):
            return message
        case .noConnection:
            return "No internet connection. Please try again."
        case .decodingError:
            return "The server returned an unexpected response."
        }
    }
}
