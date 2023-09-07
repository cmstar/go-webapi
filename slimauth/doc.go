/*
slimauth 实现 SlimAuth 协议，它是带有签名校验逻辑的 SlimAPI 的扩展。

但是和 SlimAPI 相比，当前不支持 multipart/form-data 类型的请求。

# 签名校验

每个 API 调用者会被分配到一组配对的 key-secret ， key 用于标识调用者的身份， secret 用于生成签名。
在发起 HTTP 请求时，签名信息放在 Authorization 头，格式为：

	Authorization: SLIM-AUTH Key={key}, Sign={sign}, Timestamp={timestamp}, Version=1

花括号内是可变的参数值。除开头的 scheme 部分外，其余各参数由逗号隔开，顺序不做要求，参数名称前的空白字符会被忽略。各参数定义为:
  - Authorization scheme 固定为 SLIM-AUTH 。
  - Key 是请求者的 key 。
  - Sign 是基于请求内容和 secret 生成的签名。详见签名算法节。
  - Timestamp 是生成签名时的 UNIX 时间戳，单位是秒。
  - Version 表示签名算法的版本，当前固定值为 1 。可省略，省略时默认为 1 。

API 服务器将根据签名算法，校验 Sign 的值是否正确，并要求 Timestamp 在允许的误差范围内（默认为 300 秒）。

特别的，当不方便定制请求头时，也可以将 Authorization 头的值，放在 URL 的 ~auth 参数上（记得 urlEncode ）。
~auth 参数不参与签名计算。如果同时提供参数和请求头，则只读取请求头。
此功能特别适用于 JSONP 请求（ SlimAPI 的功能之一），因为其不能定制 HTTP 头。

# 签名算法

字符集统一使用 UTF-8 。签名使用 HMAC-SHA256 算法，通过 secret 对待签名串进行哈希计算得到。待签名串根据请求的内容生成，格式为：

	TIMESTAMP
	METHOD
	PATH
	QUERY_VALUES
	BODY_VALUES (optional)
	END  (constant)

每个部分间用换行符（\n）分割，各部分的值为：
 1. TIMESTAMP 是生成签名时的 UNIX 时间戳，需和 Authorization 头里的 Timestamp 参数值一样。
 2. METHOD 是 HTTP 请求的 METHOD ，如 GET/POST/PUT 。
 3. PATH 请求的路径，没有路径部分时，使用“/”。
    比如请求地址是“http://temp.org/the/path/”则路径为“/the/path/”；
    地址是“http://temp.org/”或“http://temp.org”，路径均为“/”。
 4. QUERY_VALUES 是 URL 的 query string 部分拼接后的值。
    先按参数名称的 UTF-8 字节顺序升序，将参数排列好，需使用稳定的排序算法，这样若有同名参数，其顺序不会被打乱；
    然后排序后的参数的值紧密拼接起来（无分隔符）；
    若一个参数没有值，如“?a=&b=2”或“?a&b=2”中的“a”，则用参数名称代替值拼入。
    没有 query string 时，整个 QUERY 部分使用一个空字符串。
 5. BODY_VALUES 若是 application/x-www-form-urlencoded 请求，则处理方式同 QUERY 。
    若是 application/json 请求，则为 JSON 原文，和 BODY 上送的一致，不做任何修改。
    GET 请求时此部分省略（包含换行符均省略）。
    不支持其他类型的请求。
 6. 最后一行固定是“END”三个字符，末尾没有空行。

注意：
  - UTF-8 字节顺序不是字典顺序，字节顺序下，英文大写字母在小写字母前面，比如 X 排序在 a 前面。
  - 如果在 URL 上使用 ~auth 参数，此参数不参与签名计算。

# 例子1 - 参数的排序规则

当前示例及其后的示例中，时间戳均固定为 1662439087 ，使用的 Key 的值为 my_key ， secret 的值为 my_secret 。

待签名的请求为：

	POST http://temp.org/my/path?a&c=3&b=2&z=4&X=%E4%B8%AD%E6%96%87&a=1&b=
	Content-Type: application/x-www-form-urlencoded

	p1=11&p3=33&p2=22

这是请求的 HTTP 报文的内容（除去 HTTP/1.1 部分）。
  - 请求的 QUERY 部分为 a&c=3&b=2&z=4&X=%E4%B8%AD%E6%96%87&a=1&b= ，参数使用百分号转义（Percent-encoding）过， X 参数原始值为“中文”。
  - 请求的 BODY 部分是 p1=11&p3=33&p2=22 。

获取待签名串的步骤：
 1. 拼接 TIMESTAMP ，值为 1662439087 。
 2. 拼接 METHOD ，值为 POST 。
 3. 拼接 PATH ，即 http://temp.org/my/path 中的 /my/path 。
 4. 计算并追加 QUERY 部分。见下文描述。
 5. 计算并追加 BODY 部分。由于是 application/x-www-form-urlencoded 的请求， BODY 部分的处理和 QUERY 规则一样，结果为： "112233" 。
 6. 追加最后一行，固定值为 END 。

QUERY 部分的计算步骤为：
 1. 得到参数表 [a, c, b, z, X, a, b] ，将参数根据名称按 UTF-8 字节顺序升序排列，并且使用稳定排序算法。
    排列后为 [X, a, a, b, b, c, z] ，其中两个“a”参数和“b”参数的顺序需和 URL 中出现的顺序一致。
 2. 按排序后的参数顺序，得到参数的原始值为：[中文, , 1, 2, , 3, 4] ，其中有两个空白值（ X 和 b 参数），对应没有值的第一个“a”参数和第二个“b”参数。
 3. 按顺序将值拼接起来，若参数没有值，则使用参数名称替代，得到： "中文a12b34" 。

最终待签名串为：

	1662439087
	POST
	/my/path
	中文a12b34
	112233
	END

通过 my_secret 计算 HMAC-SHA256 值为： b3baa63839877585cc05495810fb10267317df2fceda2eddcb92a740f78d1ba5

拼接得到 Authorization 头，追加到请求头，最终请求为：

	POST http://temp.org/my/path?a&c=3&b=2&z=4&X=%E4%B8%AD%E6%96%87&a=1&b=
	Content-Type: application/x-www-form-urlencoded
	Authorization: SLIM-AUTH Key=my_key, Sign=b3baa63839877585cc05495810fb10267317df2fceda2eddcb92a740f78d1ba5, Timestamp=1662439087, Version=1

	p1=11&p3=33&p2=22

# 例子2 - 空白请求

	GET http://temp.org

待签名串为：

	1662439087
	GET
	/

	END

由于是 GET 请求，待签名串由5部分构成，没有 BODY 部分；同时此请求没有参数，故 QUERY 部分为空字符串。
最终请求为：

	GET http://temp.org
	Authorization: SLIM-AUTH Key=my_key, Sign=980b8715cefc0b98ae2b0788ce849308757554fbe685a05a43e6bc31fb0d0a4c, Timestamp=1662439087, Version=1

# 例子3 - 使用 JSON 请求

	POST http://temp.org/p/?x=1&y=2
	Content-Type: application/json

	{"key":"value"}

JSON 请求的 BODY 部分不用额外处理，直接原样拼接到待待签名串。注意：如果 JSON 中本身有换行，也原样拼接，不需要额外处理。

得到待签名串为：

	1662439087
	POST
	/p/
	12
	{"key":"value"}
	END

最终请求为：

	POST http://temp.org/p/?x=1&y=2
	Content-Type: application/json
	Authorization: SLIM-AUTH Key=my_key, Sign=ce0906df79291d516bb443adbc6099b39f36c006696150202e4e41ffe7dab211, Timestamp=1662439087, Version=1
*/
package slimauth
