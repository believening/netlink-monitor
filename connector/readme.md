# 进程监控

## netlink connector 报文消息

netlink connector 报文需要组装出如下内容：

```
---------------------------------
|  nlmsghdr |  cn_msg  |  data  |
---------------------------------
```

对应的数据结构定义如下：

``` c
struct nlmsghdr {
	__u32		nlmsg_len;	/* Length of message including header */
	__u16		nlmsg_type;	/* Message content */
	__u16		nlmsg_flags;	/* Additional flags */
	__u32		nlmsg_seq;	/* Sequence number */
	__u32		nlmsg_pid;	/* Sending process port ID */
};

/*
 * idx and val are unique identifiers which 
 * are used for message routing and 
 * must be registered in connector.h for in-kernel usage.
 */

struct cb_id {
	__u32 idx; // 对于进程监控来说 idx 和 val 的值都是 1，已经通过宏定义了
	__u32 val; // 分别是 CN_IDX_PROC 和 CN_VAL_PROC
};

struct cn_msg {
	struct cb_id id;

	__u32 seq;
	__u32 ack;

	__u16 len;		/* Length of the following data */
	__u16 flags;
	__u8 data[0];
};
```

需要注意的是 `nlmsghdr.nlmsg_len` 所指的长度是包含了`nlmsghdr`自身、`cn_msg`、以及消息内容本身的字节长度。 而 `cn_msg.len` 仅仅指的是消息内容本身的长度。

## 参考链接

* [connector](https://www.kernel.org/doc/Documentation/connector/connector.txt)
* [connect.h](https://code.woboq.org/linux/linux/include/linux/connector.h.html)
* [cn_proc.h](https://code.woboq.org/linux/linux/include/linux/cn_proc.h.html)
* [连接器（Netlink Connector）及其应用](https://www.ibm.com/developerworks/cn/linux/l-connector/)
* [linux process monitoring](https://bewareofgeek.livejournal.com/2945.html)