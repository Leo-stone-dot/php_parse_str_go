# php_parse_str_go

### 生成说明（Provenance）
- 本仓库代码由 ByteDance DevInfra 的 DevAgent/AIME 生成
- 生成日期：2025-09-30
- 当前维护者：zhaixiaolei.leo

一个纯标准库实现的 Go 包，尽可能贴近 PHP `parse_str` 的语义：
- 输入为原始查询串（支持以 `?` 开头）
- 参数分隔符支持 `&` 和 `;`（对齐 PHP 默认 `arg_separator.input` 行为）
- 键值对仅在第一个 `=` 处分割；无 `=` 的键，其值为空字符串
- 使用 `application/x-www-form-urlencoded` 的解码规则：
  - `+` 解码为空格
  - `%XX` 按十六进制解码；当序列不合法时（如缺位、非十六进制），在默认模式下不报错，保留原文
- 支持 PHP 风格的括号语法构造数组/对象：
  - `key[]=v` 依次追加到数组
  - `key[0]=v` 数组按索引设置，必要时以 `nil` 填充空洞
  - `key[sub]=v` 进入子映射 `map[string]any`
  - 嵌套：`a[b][c]=d`、`a[][b]=c`、`a[0][1]=x` 等
- 重复键策略：
  - 纯标量（无括号）按“最后一次赋值生效”
  - 使用 `[]` 追加的数组按出现顺序累积
  - 标量后紧跟 `[]` 的情况（如 `a=1&a[]=2`）：把先前标量转成数组首元素 `"1"`，然后继续追加
  - 重复关联键（如 `a[b]=x&a[b]=y`）在叶子上最后一次赋值生效
- 叶子值类型为 `string`；容器为 `map[string]any` 或 `[]any`；数值索引的空洞以 `nil` 表示
- 鲁棒性：忽略完全空的片段；对解码失败宽容；解码后会对键值做首尾空白裁剪

## 安装与使用

```shell
go get github.com/Leo-stone-dot/php_parse_str_go
```


示例：

```go
package main

import (
    "fmt"
    parsephp "php_parse_str_go/parsephp"
)

func main() {
    result, err := parsephp.ParseStr("a[]=1&a[]=2&a[b]=x")
    if err != nil {
        panic(err)
    }
    fmt.Printf("%#v\n", result)
}
```

输出结构（示意）：

```
map[string]interface {}{
  "a": []interface {}{"1", "2", map[string]interface {}{"b": "x"}},
}
```

## API

- `ParseStr(query string) (map[string]any, error)`
  - 使用默认选项解析
- `ParseStrWithOptions(query string, opts Options) (map[string]any, error)`
  - 可配置解析行为

### Options 与默认值

```go
// Separators: 用于分隔参数的字符，默认 ['&', ';']
// StrictDecode: 为 true 时，遇到非法百分号转义将返回错误；
//               为 false 时（默认），非法转义会被原样保留，不影响整体解析。
type Options struct {
    Separators   []rune
    StrictDecode bool
}

var DefaultOptions = Options{
    Separators:   []rune{'&', ';'},
    StrictDecode: false,
}
```

## 单元测试覆盖的语义片段

- `a=b&a=c` -> `{ "a": "c" }`
- `a[]=b&a[]=c` -> `{ "a": ["b", "c"] }`
- `a[0]=b&a[2]=c` -> `{ "a": ["b", nil, "c"] }`
- `a[b][c]=d&a[b][e]=f` -> `{ "a": {"b": {"c": "d", "e": "f"}} }`
- `a[][b]=c&a[][b]=d` -> `{ "a": [{"b": "c"}, {"b": "d"}] }`
- `a=1&a[]=2&a[]=3` -> `{ "a": ["1", "2", "3"] }`
- `;a=b;c=d` -> `{ "a": "b", "c": "d" }`
- `q=%2B+%2520` -> `{ "q": "+ %20" }`（`+` -> 空格；`%25` -> `%`，不做递归解码）
- `flag` -> `{ "flag": "" }`
- `?x=1&y=2` -> `{ "x": "1", "y": "2" }`
- `a[0][1]=x` -> `{ "a": [[nil, "x"]] }`
- `a[b]=x&a[b]=y` -> `{ "a": {"b": "y"} }`

## 设计与实现要点

- `tokenizeKey(s string) []string`：返回 `base + tokens`，其中空括号记为 `""`
- `decode(s string, strict bool) (string, error)`：优先用 `url.QueryUnescape`，失败时在非严格模式下使用自实现的宽容解码（仅解合法 `%XX`，非法 `%` 保留原文）
- 容器决策：
  - 首个 token 为空或数字 -> 选择 `[]any`
  - 首个 token 为非数字字符串 -> 选择 `map[string]any`
  - 标量 -> 最后一次赋值覆盖；若随后进入 `[]`，把标量提升为首元素
- `growSlice([]any, idx int)`：扩容并以 `nil` 填充至所需索引
- 冲突解决：
  - 遇到不匹配的容器类型时，依据下一个 token 的类型替换为合适的容器（保证不 panic，且尽量稳定）

## 兼容性与差异

- PHP 对变量名为空字符串有历史包袱；本实现会忽略空键以保持结构可用性
- 当标量后跟 `key[sub]`（进入映射）时，先前标量会被抛弃以与 PHP 行为保持一致；当标量后跟 `key[]`（进入数组）时，先前标量会转为数组首元素
- 百分号转义的递归（例如 `%2520` 仅解码一次得到 `%20`）与浏览器常见行为一致

## 运行测试

在项目根目录执行：

```bash
go vet ./...
go test ./...
```

## 边界行为澄清（括号与混合容器）

- 不匹配的左括号 `[`：在键的基名（base）中转换为下划线 `_`，其后字符按字面保留。例如：`a[=1` → `{ "a_": "1" }`，`p[q=1` → `{ "p_q": "1" }`。
- 成对括号后多余的右括号 `]`：在紧随匹配对关闭后出现的额外 `]` 将被忽略。例如：`a[b]]=1` → `{ "a": {"b": "1"} }`。
- 游离的右括号 `]`（不在括号 token 解析中）：作为字面量保留在基名中。例如：`b]=1` → `{ "b]": "1" }`，`x]=1` → `{ "x]": "1" }`。
- 混合容器（map/slice）在同一 base 下的稳定语义：
  - 若 base 已是映射（例如先有 `a[b]=x`），随后 `a[]=y`、`a[]=z` 会在映射下以数字字符串键追加：`{"a": {"b":"x","0":"y","1":"z"}}`。
  - 若 base 先为切片（例如 `a[]=x`），随后出现关联键（`a[b]=z`）会将切片保留式转换为映射（元素转为字符串索引键），如：`{"a": {"0":"x","b":"z"}}`。
- token 内的边缘解码：对键名而言，先按原始字符串分割括号 token，再分别对 base 与各 token 进行解码；因此编码的括号在 token 内容内按字面处理而不会改变结构边界。例如：`a[%5D]=x` → `{ "a": {"]": "x"} }`，`a[%5B]=y` → `{ "a": {"[": "y"} }`。
