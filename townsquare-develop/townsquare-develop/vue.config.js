module.exports = {
  // if the app is supposed to run on Github Pages in a subfolder, use the following config:
  // publicPath: process.env.NODE_ENV === "production" ? "/townsquare/" : "/"
  publicPath: process.env.NODE_ENV === "production" ? "/" : "/",
  lintOnSave: false,  // 临时禁用ESLint以快速启动
  devServer: {
    host: '0.0.0.0',  // 监听所有网络接口，允许局域网访问
    port: 8081,
    allowedHosts: 'all'  // 允许所有主机访问（替代disableHostCheck）
  }
};
