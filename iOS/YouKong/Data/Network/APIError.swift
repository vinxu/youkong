import Foundation

enum APIError: Error, LocalizedError {
    case invalidURL
    case invalidResponse
    case decodingError(Error)
    case networkError(Error)
    case serverError(code: Int, message: String)
    case unauthorized
    case tokenExpired
    case notFound
    case forbidden
    case parameterError(String)
    case unknown

    var errorDescription: String? {
        switch self {
        case .invalidURL:
            return "无效的请求地址"
        case .invalidResponse:
            return "服务器响应无效"
        case .decodingError:
            return "数据解析失败"
        case .networkError(let error):
            return "网络错误: \(error.localizedDescription)"
        case .serverError(_, let message):
            return message
        case .unauthorized:
            return "请先登录"
        case .tokenExpired:
            return "登录已过期，请重新登录"
        case .notFound:
            return "资源不存在"
        case .forbidden:
            return "无权限访问"
        case .parameterError(let message):
            return message
        case .unknown:
            return "未知错误"
        }
    }

    static func from(code: Int, message: String) -> APIError {
        switch code {
        case 0:
            return .unknown
        case 1001:
            return .parameterError(message)
        case 1002:
            return .unauthorized
        case 1003:
            return .tokenExpired
        case 1004:
            return .notFound
        case 1005:
            return .forbidden
        case 5000:
            return .serverError(code: code, message: message)
        default:
            return .serverError(code: code, message: message)
        }
    }
}
