/*
Package slimapi 基于 webapi 包，实现基于 SlimAPI 协议的开发框架。

SlimAPI 是一个基于 HTTP WebAPI 的通信契约。旨在将代码与 HTTP 通信解耦，使编码者可以更多的关注业务逻辑而不是通信方式。

SlimAPI 请求

可以通过 HTTP 的 Content-Type 头指定使用何种格式请求，目前支持的类型如下：
	- GET 不读取 Content-Type 头。
	- POST FORM 表单格式， Content-Type 可以是 application/x-www-form-urlencoded 或 multipart/form-data 。
	- POST JSON 以 JSON 作为数据，值为 application/json 。

也可以不指定 Content-Type 头，而通过`~format`参数指定格式，详见下文。

URL 形式1

http://domain/ApiEntry?~method=METHOD&~format=FORMAT&~callback=CALLBACK

以“~”标记的参数为 API 框架的元参数：
	- ~method：必填；表示被调用的方法的名称。
	- ~format：可选；请求所使用的数据格式，支持get/post/json；此参数在可以在不方面指定`Content-Type`时提供相同的功能。
	- ~callback：可选；JSONP回调函数的名称，一旦制定此参数，返回一个 JSONP 结果，Content-Type: text/javascript 。

参数名称都是大小写不敏感的。`~format`参数优先级高于`Content-Type`，若指定了`~format`，则`Content-Type`的值被忽略。

`~format`的可选值：
	- get 默认值。使用 GET 方式处理。
	- post 效果等同于给定 Content-Type: application/x-www-form-urlencoded
	- json 效果等同于给定 Content-Type: application/json

URL 形式2

http://domain/ApiEntry?METHOD.FORMAT(CALLBACK)
不需要再写参数名字，直接将需要的元参数值追加在URL后面。

同形式1，“.FORMAT”和“(CALLBACK)”是可选的，
省略“.FORMAT”后形如： http://domain/ApiEntry?METHOD(CALLBACK) ；
省略“(CALLBACK)”后形如： http://domain/ApiEntry?METHOD.FORMAT 。

URL 形式3

通过路由规则，将元参数编排到 URL 路径里。这是最常见的方案： http://domain/ApiEntry/METHOD
这里 METHOD 就是元参数 ~method 。


SlimAPI 请求参数的格式

请求参数
- GET 参数体现在 URL 上，形如 data=1&name=abc&time=2014-4-8 。
- 表单 以 POST 方式放在HTTP BODY中。
- JSON 只能使用 POST 方式上送。

可在 GET/表单参数中传递简单的数组，数组元素间使用 ~ 分割，如 1~2~3~4~5 可表示数组 [1, 2, 3, 4, 5]。

在GET/表单格式下：

	data=1&name=abc&time=2014-4-8&array=1~2~3~4

与JSON格式下的下面内容等价：

	{ "data":1, "name": "abc", "time": "2014-4-8", "array": [1, 2, 3, 4] }

日期格式使用字符串的 yyyy-MM-dd HH:mm:ss 格式，默认为 UTC 时间。也支持 RFC3339 ，这种格式自带时区。


SlimAPI 回执格式

若指定了 ~callback 参数，则返回结果为 JSONP 格式： Content-Type: text/javascript ；否则为 JSON 格式： Content-Type: application/json 。

状态码总是200，具体异常码需要从Code字段判定。数据装在一个基本的信封中，信封格式如下：

	{ Code: 0, Message: "", Data: {} }

	- Code 0为API调用成功未见异常，非0值为异常：
		- 1-999 API请求、通信及运行时异常，尽可能与 HTTP 状态码一致：
			- 500 服务端内部异常。
			- 400 请求参数或报文错误。
			- 403 客户端无访问权限。
		- 其他约定：
		- -1 未明确定义的错误。
		- 1000-9999 预留
		- 大于等于10000为业务预定义异常，由具体API自行定义，但建议至少分为用户可见和不可见两个区间：
			- 10000-19999 建议保留为不提示给用户的业务异常，对接 API 的客户端对于这些异常码，对用户提示一个统一的如“网络异常”的错误。
			- 20000-29999 对接 API 的客户端可直接将 Message 展示给用户，用于展示的错误消息需要由服务端控制的场景。
	- Message 附加信息， Code 不为0时记录错误描述，可为空字符串。
	- Data 返回的主数据，不同API各不相同，其所有可能形式如下：
		- 若API没有返回值，则为 null 。
		- 对于返回布尔型结果的 API ，Data 为 true 或 false 。
		- 对于返回数值结果的 API ，Data即为数值，如：123.654 。
		- 对于返回字符串结果的 API ， Data 为字符串，如："string value" 。
		- 日期也作为字符串，参照前面提到的日期格式。
		- 对于返回集合的 API ， Data 为数组，如： ["result1", "result2"] 。
		- 对于返回复杂对象的 API ，Data 为 JSON object ，如：{ "Field1": "Value1", "Field2": "Value2" } 。
*/
package slimapi
