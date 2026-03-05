module.exports = {
  // if the app is supposed to run on Github Pages in a subfolder, use the following config:
  // publicPath: process.env.NODE_ENV === "production" ? "/townsquare/" : "/"
  publicPath: process.env.NODE_ENV === "production" ? "/" : "/",
  devServer: {
    port: 8092,
    // 如果 8092 被占用则自动尝试下一个端口
    // Windows Hyper-V 会排除 8080-8090 范围，故默认用 8092
  }
};
