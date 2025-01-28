# Client for Gnome

## 特性

1. 使用 xprop+xdotool 获取前台软件名（WMClass+WMName）
2. 通过 dbus 监听休眠（及恢复）、关机事件，自动修改状态
3. 支持替换软件名来~~搞点大新闻~~

## 快速使用

1. 仿照 `.env.example` 创建一个 `.env` 文件
2. 运行 `go run client/gnome`

## 小寄巧：替换软件名

`patterns.go` 中可以自定义一些软件的显示内容，如果不提供 message 的话就会显示成`正在使用「${WMName}」`

## 错误排查

如果你通过 systemd 来运行它的话，xdotool 就会以非零状态码退出（就是出错了），搜索了一番，找到了解决方法，需要在 service 文件里加上：

```diff
[Service]
Type=simple
+ Environment="DISPLAY=:1" // 通过 echo $DISPLAY 获取
ExecStart=/home/roitium/Applications/my-status/main
Restart=always
+ User=your-user-name // 你当前用户的用户名和组（不是 root 的）
+ Group=your-user-group
RestartSec=10
WorkingDirectory=/home/roitium/Applications/my-status
+ Environment="XAUTHORITY=/home/your-user-name/.Xauthority"
```
