// http.ts - 基础HTTP客户端

/**
 * HTTP错误类
 * 用于表示HTTP请求失败的错误
 */
class HTTPError extends Error {
  status: number
  statusText: string

  /**
   * 创建HTTP错误实例
   * @param status HTTP状态码
   * @param statusText 状态文本
   * @param message 错误消息
   */
  constructor(status: number, statusText: string, message?: string) {
    super(message || statusText)
    this.status = status
    this.statusText = statusText
  }
}

/**
 * 发送HTTP请求
 * @param url 请求URL
 * @param options 请求选项
 * @returns 响应数据
 */
async function request<T>(url: string, options: RequestInit = {}): Promise<T> {
  const defaultOptions: RequestInit = {
    credentials: 'include',
  }

  const response = await fetch(url, { ...defaultOptions, ...options })

  if (!response.ok) {
    throw new HTTPError(
      response.status,
      response.statusText,
      `HTTP ${response.status}`,
    )
  }

  // 处理空响应（例如204 No Content）
  if (response.status === 204) {
    return {} as T
  }

  const contentType = response.headers.get('content-type')
  if (contentType && contentType.includes('application/json')) {
    return response.json()
  }
  return response.text() as unknown as T
}

export const http = {
  /**
   * 发送GET请求
   * @param url 请求URL
   * @param options 请求选项
   * @returns 响应数据
   */
  get: <T>(url: string, options?: RequestInit) =>
    request<T>(url, { ...options, method: 'GET' }),
  /**
   * 发送POST请求
   * @param url 请求URL
   * @param body 请求体
   * @param options 请求选项
   * @returns 响应数据
   */
  post: <T>(url: string, body?: any, options?: RequestInit) => {
    const headers: HeadersInit = { ...options?.headers }
    let requestBody = body

    if (
      body &&
      typeof body === 'object' &&
      !(body instanceof FormData) &&
      !(body instanceof Blob)
    ) {
      if (!('Content-Type' in headers)) {
        ;(headers as any)['Content-Type'] = 'application/json'
      }
      requestBody = JSON.stringify(body)
    }

    return request<T>(url, {
      ...options,
      method: 'POST',
      headers,
      body: requestBody,
    })
  },
  /**
   * 发送PUT请求
   * @param url 请求URL
   * @param body 请求体
   * @param options 请求选项
   * @returns 响应数据
   */
  put: <T>(url: string, body?: any, options?: RequestInit) => {
    const headers: HeadersInit = { ...options?.headers }
    let requestBody = body

    if (
      body &&
      typeof body === 'object' &&
      !(body instanceof FormData) &&
      !(body instanceof Blob)
    ) {
      if (!('Content-Type' in headers)) {
        ;(headers as any)['Content-Type'] = 'application/json'
      }
      requestBody = JSON.stringify(body)
    }

    return request<T>(url, {
      ...options,
      method: 'PUT',
      headers,
      body: requestBody,
    })
  },
  /**
   * 发送DELETE请求
   * @param url 请求URL
   * @param options 请求选项
   * @returns 响应数据
   */
  delete: <T>(url: string, options?: RequestInit) =>
    request<T>(url, { ...options, method: 'DELETE' }),
}
