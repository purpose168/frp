import { http } from './http'
import type { StatusResponse } from '../types/proxy'

/**
 * 获取frpc状态
 * @returns 状态响应
 */
export const getStatus = () => {
  return http.get<StatusResponse>('/api/status')
}

/**
 * 获取frpc配置
 * @returns 配置内容
 */
export const getConfig = () => {
  return http.get<string>('/api/config')
}

/**
 * 更新frpc配置
 * @param content 新的配置内容
 */
export const putConfig = (content: string) => {
  return http.put<void>('/api/config', content)
}

/**
 * 重新加载frpc配置
 */
export const reloadConfig = () => {
  return http.get<void>('/api/reload')
}
