#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/socket.h>
#include <linux/netlink.h>
#include <linux/cn_proc.h>
#include <linux/connector.h>
#include <unistd.h>
#include <signal.h>

#define CNMSG_LEN sizeof(struct cn_msg)
#define CNMCASTOP_LEN sizeof(enum proc_cn_mcast_op)
#define PROCEVENT_LEN sizeof(struct proc_event)

int seq = 0;

// 创建连接
static int create_connect()
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
    l_local.nl_groups = 1;
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
static int control_event_listenning(int sd, enum proc_cn_mcast_op op)
{
    int ret;
    int msg_len;
    struct nlmsghdr nlh = {}; // netlink 消息头
    struct cn_msg cn = {};    // netlink connector 消息
    enum proc_cn_mcast_op alloc_op;
    enum proc_cn_mcast_op *o;

    msg_len = NLMSG_SPACE(CNMSG_LEN + CNMCASTOP_LEN); // 申请对齐空间
    memset(&nlh, 0, msg_len);

    nlh.nlmsg_len = msg_len;
    nlh.nlmsg_type = NLMSG_DONE;
    nlh.nlmsg_flags = 0;
    nlh.nlmsg_seq = seq;
    nlh.nlmsg_pid = getpid();

    cn.id.idx = CN_IDX_PROC;
    cn.id.val = CN_VAL_PROC;
    cn.len = CNMCASTOP_LEN;
    cn.seq = seq++;
    cn.ack = 0;                           // 对于非响应消息，ack 应当设置为 0
    o = (enum proc_cn_mcast_op *)cn.data; // 取得 connector 消息中数据部分的首地址
    *o = op;                              // 将控制模式写入到 connector 消息的数据部分中

    memcpy(NLMSG_DATA(&nlh), &cn, CNMSG_LEN + cn.len); // 填充 nlh 的 cn_msg

    ret = send(sd, &nlh, nlh.nlmsg_len, 0); // 发送单个消息使用 send() 函数
    if (ret == -1)
    {
        perror("netlink connector send");
    }
    return ret;
}

// 接受并处理事件
static int handle_received_event(int sd)
{
    int ret;
    int msg_len;
    // 连续分配在栈区的内存
    struct nlmsghdr nlh = {};   // 栈区
    struct cn_msg cn = {};      // 栈区
    struct proc_event evt = {}; // 栈区 用于申请空间
    struct proc_event *ev;

    msg_len = NLMSG_SPACE(CNMSG_LEN + PROCEVENT_LEN);
    memset(&nlh, 0, msg_len);

    ret = recv(sd, &nlh, msg_len, 0);
    if (ret == 0) // 接收了 0 个字节的内容
    {
        return 0;
    }
    else if (ret == -1)
    {
        perror("recv");
        return -1;
    }
    ev = (struct proc_event *)cn.data;
    printf("nlh: %p\n", &nlh);         // size 16;   0x7ffd558ce410
    printf("cn: %p\n", &cn);           // size 20+1; 0x7ffd558ce420 偏移 16 字节
    printf("cn.data: %p\n", &cn.data); // size 1;    0x7ffd558ce434 偏移 16 + 20 字节
    printf("proc: %p\n", &evt);        // size XX;   0x7ffd558ce440　偏移　16 + 20 + 12 字节，12个字节来自于对齐字节
    printf("size of proc evt: %ld\n", sizeof(struct proc_event));

    switch (ev->what)
    {
    case PROC_EVENT_EXEC:
        printf("process event:\texec\t[time:%lld,pid:%d,tgid:%d]\n",
               ev->timestamp_ns,
               ev->event_data.exec.process_pid,
               ev->event_data.exec.process_tgid);
        break;
    case PROC_EVENT_EXIT:
        printf("process event:\texit\t[time:%lld,pid:%d,tgid:%d,exit:%d]\n",
               ev->timestamp_ns,
               ev->event_data.exit.process_pid,
               ev->event_data.exit.process_tgid,
               ev->event_data.exit.exit_code);
        break;
    default:
        printf("process unmonitor event:\t%d\n", ev->what);
        break;
    }
    return 0;
}

int main(int argc, char *argv[])
{
    int sd; // socket fd
    int ret;

    // connect
    sd = create_connect();
    if (sd == -1)
    {
        perror("connect");
        return -1;
    }

    // register listenning
    if (control_event_listenning(sd, PROC_CN_MCAST_LISTEN) == -1)
    {
        perror("register listen listenning");
        ret = -1;
        goto out;
    }

    // handler event
    while (1)
    {
        ret = handle_received_event(sd);
        if (ret == -1)
        {
            perror("handle received event");
            ret = -1;
            goto out;
        }
    }

out:
    close(sd);
    return ret;
}