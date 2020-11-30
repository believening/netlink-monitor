# 字节对齐

## c 语言

* 结构体内的成员地址起始的位置与结构体首地址的差(即相对与该结构体首地址的偏移量)应该为该成员对齐值的整数倍。
* 结构体的对齐值为所有成员对齐值的最大值 s.AL = max{s.a.AL,s.b.AL,...}，包括嵌套结构体中的成员。
* 结构体的末尾会被填充到其对齐值的整数倍。

* 高级对齐控制
  
  - __attribute__((alignd(SIZE))) 成员之间以 SIZE 大小对齐
  - __attribute__((packed)) 成员之间紧密排列

## go语言

同 c 语言类似，但是[无高级对齐控制属性](https://github.com/golang/go/wiki/cgo#struct-alignment-issues)
