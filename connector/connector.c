#include <stdio.h>
#include <string.h>
#include <sys/socket.h>
#include <linux/netlink.h>
#include <linux/cn_proc.h>
#include <linux/connector.h>
#include <sys/types.h>
#include <unistd.h>

#define CNMSG_LEN sizeof(struct cn_msg)
#define CNMCASTOP_LEN sizeof(enum proc_cn_mcast_op)

int seq = 0;

void change_cn_proc_mode(int mode)
{
    struct nlmsghdr *nlhdr = NULL;
    memset(nlhdr, 0, NLMSG_HDRLEN + sizeof(int));
}

// 创建连接
int connect()
{
    int s; // socket fd
    struct sockaddr_nl l_local;

    // socket
    s = socket(AF_NETLINK, SOCK_DGRAM, NETLINK_CONNECTOR);
    if (s == -1)
    {
        perror("socket");
        return -1;
    }

    // bind
    l_local.nl_family = AF_NETLINK;
    l_local.nl_groups = 0;
    l_local.nl_pid = getpid();

    if (bind(s, (struct sockaddr *)&l_local, sizeof(l_local)) != 0)
    {
        perror("bind");
        close(s);
        return -1;
    }
    return s;
}

// 控制事件侦听
int control_event_listenning(int s, enum proc_cn_mcast_op op)
{
    int ret;
    struct nlmsghdr *hdr = NULL; // netlink 消息头
    struct cn_msg *cn = NULL;    // netlink connector 消息
    // struct msghdr *msg = NULL;
    enum proc_cn_mcast_op *o;

    memset(hdr, 0, NLMSG_HDRLEN);
    hdr->nlmsg_len = NLMSG_LENGTH(CNMSG_LEN + CNMCASTOP_LEN);
    hdr->nlmsg_type = NLMSG_DONE;
    hdr->nlmsg_flags = 0;
    hdr->nlmsg_seq = seq;
    hdr->nlmsg_pid = getpid();

    cn = (struct cn_msg *)NLMSG_DATA(hdr); // 取得 netlink 消息中数据部分的首地址
    cn->id.idx = CN_IDX_PROC;
    cn->id.val = CN_VAL_PROC;
    cn->len = CNMCASTOP_LEN;
    cn->seq = seq++;
    cn->ack = 0;
    o = (enum proc_cn_mcast_op *)cn->data; // 取得 connector 消息中数据部分的首地址
    *o = op;                               // 将控制模式写入到 connector 消息的数据部分中

    ret = send(s, hdr, hdr->nlmsg_len, 0);
    if (ret != 0)
    {
        perror("netlink connector send");
        return -1;
    }
    return ret;
}
//
int main(int argc, char *argv[])
{
    int s; // socket fd
    struct sockaddr_nl l_local;

    // connect
    s = connect();
    if (s == -1)
    {
        perror("connect");
        return -1;
    }

    // register listenning
    if (control_event_listenning(s, PROC_CN_MCAST_LISTEN) != 0)
    {
        perror("listen listenning");
        return -1;
    }
}