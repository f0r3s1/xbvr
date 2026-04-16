import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import path from 'path'
import { readFileSync, writeFileSync } from 'fs'

// videojs-vr 2.0.0 bundles THREE.js r125 where SphereGeometry extends BufferGeometry,
// but changeProjection_ still uses the old Geometry API (faceVertexUvs).
// Patches those calls with the modern BufferGeometry attributes.uv API.
// Uses explicit '\n' concatenation so indentation is unambiguous regardless of editor.
function patchVrCode(code) {
  // 360_LR/360_TB left-eye UV
  code = code.replace(
    'let uvs = geometry.faceVertexUvs[0];\n' +
    '      for (let i = 0; i < uvs.length; i++) {\n' +
    '        for (let j = 0; j < 3; j++) {\n' +
    '          if (projection === \'360_LR\') {\n' +
    '            uvs[i][j].x *= 0.5;\n' +
    '          } else {\n' +
    '            uvs[i][j].y *= 0.5;\n' +
    '            uvs[i][j].y += 0.5;\n' +
    '          }\n' +
    '        }\n' +
    '      }\n' +
    '      this.movieGeometry = new BufferGeometry().fromGeometry(geometry);',
    '{ const uv = geometry.attributes.uv;\n' +
    '        for (let i = 0; i < uv.count; i++) {\n' +
    '          if (projection === \'360_LR\') { uv.setX(i, uv.getX(i) * 0.5); }\n' +
    '          else { uv.setY(i, uv.getY(i) * 0.5 + 0.5); }\n' +
    '        } }\n' +
    '      this.movieGeometry = geometry;'
  )

  // 360_LR/360_TB right-eye UV
  code = code.replace(
    'uvs = geometry.faceVertexUvs[0];\n' +
    '      for (let i = 0; i < uvs.length; i++) {\n' +
    '        for (let j = 0; j < 3; j++) {\n' +
    '          if (projection === \'360_LR\') {\n' +
    '            uvs[i][j].x *= 0.5;\n' +
    '            uvs[i][j].x += 0.5;\n' +
    '          } else {\n' +
    '            uvs[i][j].y *= 0.5;\n' +
    '          }\n' +
    '        }\n' +
    '      }\n' +
    '      this.movieGeometry = new BufferGeometry().fromGeometry(geometry);',
    '{ const uv = geometry.attributes.uv;\n' +
    '        for (let i = 0; i < uv.count; i++) {\n' +
    '          if (projection === \'360_LR\') { uv.setX(i, uv.getX(i) * 0.5 + 0.5); }\n' +
    '          else { uv.setY(i, uv.getY(i) * 0.5); }\n' +
    '        } }\n' +
    '      this.movieGeometry = geometry;'
  )

  // 180/180_LR left-eye UV
  code = code.replace(
    'let uvs = geometry.faceVertexUvs[0];\n' +
    '      if (projection !== \'180_MONO\') {\n' +
    '        for (let i = 0; i < uvs.length; i++) {\n' +
    '          for (let j = 0; j < 3; j++) {\n' +
    '            uvs[i][j].x *= 0.5;\n' +
    '          }\n' +
    '        }\n' +
    '      }\n' +
    '      this.movieGeometry = new BufferGeometry().fromGeometry(geometry);',
    'if (projection !== \'180_MONO\') {\n' +
    '        const uv = geometry.attributes.uv;\n' +
    '        for (let i = 0; i < uv.count; i++) { uv.setX(i, uv.getX(i) * 0.5); }\n' +
    '      }\n' +
    '      this.movieGeometry = geometry;'
  )

  // 180/180_LR right-eye UV
  code = code.replace(
    'uvs = geometry.faceVertexUvs[0];\n' +
    '      for (let i = 0; i < uvs.length; i++) {\n' +
    '        for (let j = 0; j < 3; j++) {\n' +
    '          uvs[i][j].x *= 0.5;\n' +
    '          uvs[i][j].x += 0.5;\n' +
    '        }\n' +
    '      }\n' +
    '      this.movieGeometry = new BufferGeometry().fromGeometry(geometry);',
    '{ const uv = geometry.attributes.uv;\n' +
    '        for (let i = 0; i < uv.count; i++) { uv.setX(i, uv.getX(i) * 0.5 + 0.5); } }\n' +
    '      this.movieGeometry = geometry;'
  )

  // Fix deprecated videojs.mergeOptions → videojs.obj.merge
  code = code.replaceAll('videojs.mergeOptions(', 'videojs.obj.merge(')

  // Fix deprecated videojs.bind → native Function.prototype.bind
  code = code.replace(/videojs\.bind\(([^,]+),\s*([^)]+)\)/g, '$2.bind($1)')

  // Remove THREE.js r125 deprecation console.warns (APIs still work, just noisy)
  code = code.replaceAll("console.warn('THREE.Material: .overdraw has been removed.');", '')
  code = code.replaceAll("console.warn('WebGLRenderer: .getsize() now requires a Vector2 as an argument');", '')
  code = code.replaceAll("console.warn('THREE.Quaternion: .inverse() has been renamed to invert().');", '')

  // Remove overdraw: true from material constructors (property removed in THREE.js r125)
  code = code.replace(/\s*overdraw:\s*true,?/g, '')

  return code
}

// Patches videojs-vr on disk at Vite startup so pre-bundling picks up the fix.
// Also runs as a transform hook for production builds. Idempotent.
function videojsVrGeometryFix() {
  return {
    name: 'videojs-vr-geometry-fix',
    configResolved() {
      for (const name of ['videojs-vr.es.js', 'videojs-vr.cjs.js']) {
        const file = path.resolve(__dirname, '..', 'node_modules/videojs-vr/dist', name)
        try {
          const original = readFileSync(file, 'utf8')
          const patched = patchVrCode(original)
          if (patched !== original) {
            writeFileSync(file, patched)
            console.log(`[videojs-vr-geometry-fix] patched ${name}`)
          }
        } catch { /* file missing — skip */ }
      }
    },
    transform(code, id) {
      if (!id.includes('videojs-vr') || !id.endsWith('.js')) return
      return patchVrCode(code)
    }
  }
}

export default defineConfig({
  plugins: [
    vue({
      template: {
        transformAssetUrls: {
          includeAbsolute: false,
        },
      },
    }),
    videojsVrGeometryFix(),
  ],
  base: '/ui/',
  build: {
    outDir: 'dist',
    sourcemap: false,
    emptyOutDir: true,
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, 'src'),
    },
    extensions: ['.mjs', '.js', '.ts', '.jsx', '.tsx', '.json', '.vue'],
    dedupe: ['video.js'],
  },
  optimizeDeps: {
    include: ['video.js', 'videojs-vr', 'videojs-hotkeys'],
  },
  server: {
    port: 8080,
    host: true,
    watch: {
      usePolling: true,
      interval: 500,
    },
    hmr: {
      host: 'localhost',
      port: 8080,
    },
    proxy: {
      '/api': { target: 'http://localhost:9999', changeOrigin: true },
      '/ws': { target: 'ws://localhost:9998', ws: true, changeOrigin: true },
      '/img': { target: 'http://localhost:9999', changeOrigin: true },
      '/imghm': { target: 'http://localhost:9999', changeOrigin: true },
      '/download': { target: 'http://localhost:9999', changeOrigin: true },
      '/myfiles': { target: 'http://localhost:9999', changeOrigin: true },
    },
  },
})
