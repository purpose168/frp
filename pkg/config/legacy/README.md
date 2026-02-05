到目前为止，还没有成熟的 Go 项目能够很好地解析 `*.ini` 文件。

相比之下，我们选择了一个开源项目：`https://github.com/go-ini/ini`。

这个库帮助我们解决了大部分键值匹配问题，但仍存在一些问题，例如不支持解析 `map`。

我们在这个库的基础上添加了自己的逻辑。在当前情况下，我们需要分两步完成整个 `Unmarshal`：

* 第一步，使用 `go-ini` 完成基本参数匹配；
* 第二步，解析我们的自定义参数以实现特殊结构的解析，如 `map`、`array`。

`tag` 中的一些关键字（如 inline、extends 等）可能与 Go 中的标准库（如 `json` 和 `protobuf`）不同。有关详细信息，请参考库文档：https://ini.unknwon.io/docs/intro。
