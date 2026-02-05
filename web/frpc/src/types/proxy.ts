/**
 * 代理状态接口
 */
export interface ProxyStatus {
  /** 代理名称 */
  name: string
  /** 代理类型 */
  type: string
  /** 代理状态 */
  status: string
  /** 错误信息 */
  err: string
  /** 本地地址 */
  local_addr: string
  /** 插件名称 */
  plugin: string
  /** 远程地址 */
  remote_addr: string
  /** 其他属性 */
  [key: string]: any
}

/**
 * 状态响应类型
 */
export type StatusResponse = Record<string, ProxyStatus[]>
