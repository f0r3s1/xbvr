const webpack = require("webpack");

module.exports = {
  publicPath: '/ui',
  outputDir: 'dist',
  lintOnSave: false,

  // Dev server proxy for API calls during development
  devServer: {
    port: 8080,
    hot: true,
    liveReload: true,
    webSocketServer: 'ws',
    client: {
      // Auto-detect WebSocket URL for Codespaces/Gitpod compatibility
      webSocketURL: 'auto://0.0.0.0:0/ws',
      overlay: true
    },
    allowedHosts: 'all',
    headers: {
      'Access-Control-Allow-Origin': '*'
    },
    proxy: {
      '/api': {
        target: 'http://localhost:9999',
        changeOrigin: true
      },
      '/ws/': {
        target: 'ws://localhost:9998',
        ws: true,
        changeOrigin: true
      },
      '/img': {
        target: 'http://localhost:9999',
        changeOrigin: true
      },
      '/imghm': {
        target: 'http://localhost:9999',
        changeOrigin: true
      },
      '/download': {
        target: 'http://localhost:9999',
        changeOrigin: true
      },
      '/myfiles': {
        target: 'http://localhost:9999',
        changeOrigin: true
      }
    }
  },

  chainWebpack: config => {
    config.plugins.delete('progress')
    config.plugin('simple-progress-webpack-plugin').use(require.resolve('simple-progress-webpack-plugin'), [
      {
        format: 'minimal'
      }
    ])
  },

  configureWebpack: {
    resolve: {
        fallback: {
            buffer: require.resolve('buffer/'),
        },
    },
    plugins: [
        new webpack.ProvidePlugin({
            Buffer: ['buffer', 'Buffer'],
        }),
    ],
  },

  pluginOptions: {
    i18n: {
      locale: 'en_GB',
      fallbackLocale: 'en_GB',
      localeDir: 'locales',
      enableInSFC: false
    }
  },

}
