/*
slimauth 实现 SlimAuth 协议，它是带有签名校验逻辑的 SlimAPI 的扩展。

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

# 签名算法

字符集统一使用 UTF-8 。签名使用 HMAC-SHA256 算法，通过 secret 对待签名串进行哈希计算得到。待签名串根据请求的内容生成，格式为：

	TIMESTAMP
	METHOD
	PATH
	QUERY
	BODY (optional)
	END  (constant)

每个部分间用换行符（\n）分割，各部分的值为：
  - TIMESTAMP 是生成签名时的 UNIX 时间戳，需和 Authorization 头里的 Timestamp 参数值一样。
  - METHOD 是 HTTP 请求的 METHOD ，如 GET/POST/PUT 。
  - PATH 请求的路径，包含开头的“/”，比如请求地址是“http://temp.org/the/path/” 则路径为“/the/path/”；如果没有路径部分，使用“/”。
  - QUERY 是 URL 上的参数表，按参数名称字典顺序升序，然后将值部分紧密拼接起来（无分隔符）。没有参数时，使用一个空字符串。
  - BODY 若是 application/x-www-form-urlencoded 请求，则处理方式同 QUERY 。
    若是 application/json 请求，则为 JSON 原文，和 BODY 上送的一致，不做任何修改。
    GET 请求时此部分省略（包含换行符均省略）。
    不支持其他类型的请求。
  - 最后一行固定是“END”三个字符，末尾没有空行。

# 例子1

示例中，时间戳均为 1662439087 ， Key 为 my_key ， secret 为 my_secret 。

请求：

	POST http://temp.org/my/path?c=3&b=2&z=4&x=%E4%B8%AD%E6%96%87&a=1
	Content-Type: application/x-www-form-urlencoded

	p1=11&p3=33&p2=22

签名步骤：
 1. QUERY 部分参数为 [c, b, z, x, a] ，将参数根据名称按字典顺序升序，排列后为 [a, b, c, x, z] 。
 2. 排序后的参数的原始值（没有 urlEncode 的）为：[1, 2, 3, 中文, 4] ，按顺序拼接起来，得到： "123中文4" 。
 3. 由于是 application/x-www-form-urlencoded 的请求， BODY 部分的处理和 QUERY 规则一样，结果为： "112233"

最终待签名串为：

	1662439087
	POST
	/my/path
	123中文4
	112233
	END

通过 my_secret 计算 HMAC-SHA256 值为： 28014ad8ce604ca9a1dc18c04b011e0e9d167b9818e6a0e3bee812905863914f

拼接 Authorization 头后，最终请求为：

	POST http://temp.org/my/path?c=3&b=2&z=4&x=%E4%B8%AD%E6%96%87&a=1
	Content-Type: application/x-www-form-urlencoded
	Authorization: SLIM-AUTH Key=my_key, Sign=28014ad8ce604ca9a1dc18c04b011e0e9d167b9818e6a0e3bee812905863914f, Timestamp=1662439087, Version=1

	p1=11&p3=33&p2=22

# 例子2

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

# 例子3

	POST http://temp.org//p/?x=1&y=2
	Content-Type: application/json

	{"key":"value"}

待签名串为：

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
