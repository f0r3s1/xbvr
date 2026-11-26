// Carousel and fullscreen viewer for detail views; the view provides images, getImageURL and onCarouselImageLoad
export default {
  data () {
    return {
      carouselSlide: 0,
      carouselImagesLoaded: {},
      carouselImageRetries: {},
      carouselImageKeys: {},
      carouselImageInfo: {},
      fullscreenGallery: false,
      fullscreenIndex: 0,
      fullscreenImageLoaded: false,
      fullscreenImageInfo: null,
      fullscreenZoomed: false,
      fullscreenCanZoom: false
    }
  },

  methods: {
    closeFullscreenGallery () {
      this.fullscreenGallery = false
      this.fullscreenZoomed = false
      document.body.style.overflow = ''
      this.carouselSlide = this.fullscreenIndex
    },
    detectFormatFromBytes (bytes) {
      if (bytes.length < 12) return 'IMG'

      // AVIF: check for 'ftyp' at offset 4 and 'avif', 'avis', or 'mif1' at offset 8
      if (bytes[4] === 0x66 && bytes[5] === 0x74 && bytes[6] === 0x79 && bytes[7] === 0x70) {
        const brand = String.fromCharCode(bytes[8], bytes[9], bytes[10], bytes[11])
        if (brand === 'avif' || brand === 'avis' || brand === 'mif1') return 'AVIF'
        if (brand === 'heic' || brand === 'heix') return 'HEIC'
      }

      // JPEG: starts with FF D8 FF
      if (bytes[0] === 0xFF && bytes[1] === 0xD8 && bytes[2] === 0xFF) return 'JPEG'

      // PNG: starts with 89 50 4E 47 (‰PNG)
      if (bytes[0] === 0x89 && bytes[1] === 0x50 && bytes[2] === 0x4E && bytes[3] === 0x47) return 'PNG'

      // GIF: starts with GIF87a or GIF89a
      if (bytes[0] === 0x47 && bytes[1] === 0x49 && bytes[2] === 0x46) return 'GIF'

      // WebP: starts with RIFF....WEBP
      if (bytes[0] === 0x52 && bytes[1] === 0x49 && bytes[2] === 0x46 && bytes[3] === 0x46 &&
          bytes[8] === 0x57 && bytes[9] === 0x45 && bytes[10] === 0x42 && bytes[11] === 0x50) return 'WebP'

      return 'IMG'
    },
    fetchCarouselImageInfo (url, index) {
      // a 16 byte range request is enough for the magic number; Content-Range gives the size
      fetch(url, { 
        method: 'GET',
        headers: { 'Range': 'bytes=0-15' }
      })
        .then(response => {
          const contentRange = response.headers.get('Content-Range')
          let fileSize = null
          if (contentRange) {
            const match = contentRange.match(/\/(\d+)$/)
            if (match) {
              fileSize = parseInt(match[1])
            }
          }
          return response.arrayBuffer().then(buffer => ({ buffer, fileSize }))
        })
        .then(({ buffer, fileSize }) => {
          const bytes = new Uint8Array(buffer)
          const format = this.detectFormatFromBytes(bytes)
          const size = fileSize ? this.formatBytes(fileSize) : ''
          const info = size ? `${format} · ${size}` : format
          this.carouselImageInfo[index] = info
        })
        .catch(() => {
        })
    },
    fetchFullscreenImageInfo (url) {
      this.fullscreenImageInfo = null
      // a 16 byte range request is enough for the magic number; Content-Range gives the size
      fetch(url, { 
        method: 'GET',
        headers: { 'Range': 'bytes=0-15' }
      })
        .then(response => {
          const contentRange = response.headers.get('Content-Range')
          let fileSize = null
          if (contentRange) {
            const match = contentRange.match(/\/(\d+)$/)
            if (match) {
              fileSize = parseInt(match[1])
            }
          }
          if (!fileSize) {
            const cl = response.headers.get('Content-Length')
            if (cl) fileSize = parseInt(cl)
          }
          return response.arrayBuffer().then(buffer => ({ buffer, fileSize }))
        })
        .then(({ buffer, fileSize }) => {
          const bytes = new Uint8Array(buffer)
          const format = this.detectFormatFromBytes(bytes)
          const size = fileSize ? this.formatBytes(fileSize) : ''
          this.fullscreenImageInfo = size ? `${format} · ${size}` : format
        })
        .catch(() => {
        })
    },
    formatBytes (bytes) {
      if (bytes === 0) return '0 B'
      const k = 1024
      const sizes = ['B', 'KB', 'MB']
      const i = Math.floor(Math.log(bytes) / Math.log(k))
      return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
    },
    fullscreenNext () {
      this.fullscreenImageLoaded = false
      this.fullscreenImageInfo = null
      this.fullscreenZoomed = false
      this.fullscreenCanZoom = false
      this.fullscreenIndex = (this.fullscreenIndex + 1) % this.images.length
    },
    fullscreenPrev () {
      this.fullscreenImageLoaded = false
      this.fullscreenImageInfo = null
      this.fullscreenZoomed = false
      this.fullscreenCanZoom = false
      this.fullscreenIndex = (this.fullscreenIndex - 1 + this.images.length) % this.images.length
    },
    getCarouselImageURL (url, index) {
      const key = this.carouselImageKeys[index] || 0
      const baseUrl = this.getImageURL(url, '1200x')
      return key > 0 ? baseUrl + (baseUrl.includes('?') ? '&' : '?') + '_retry=' + key : baseUrl
    },
    getFullscreenImageURL (url) {
      return this.getImageURL(url, '0x0')
    },
    handleImageClick (event) {
      if (!this.fullscreenCanZoom) return

      const gallery = this.$refs.fullscreenGallery
      const img = this.$refs.fullscreenImage

      if (this.fullscreenZoomed) {
        this.fullscreenZoomed = false
      } else {
        const rect = img.getBoundingClientRect()
        const clickRatioX = (event.clientX - rect.left) / rect.width
        const clickRatioY = (event.clientY - rect.top) / rect.height

        this.fullscreenZoomed = true

        requestAnimationFrame(() => {
          setTimeout(() => {
            const natW = img.naturalWidth
            const natH = img.naturalHeight
            const pointX = natW * clickRatioX
            const pointY = natH * clickRatioY
            const scrollX = Math.max(0, pointX - (window.innerWidth / 2))
            const scrollY = Math.max(0, pointY - (window.innerHeight / 2))

            gallery.scrollTo(scrollX, scrollY)
          }, 50)
        })
      }
    },
    onCarouselImageError (index) {
      const retries = this.carouselImageRetries[index] || 0
      if (retries < 3) {
        this.retryCarouselImage(index)
      } else {
        this.carouselImagesLoaded[index] = true
      }
    },
    onFullscreenImageLoad () {
      this.fullscreenImageLoaded = true
      if (this.$refs.fullscreenImage) {
        const img = this.$refs.fullscreenImage
        const fitsWidth = img.naturalWidth <= window.innerWidth * 0.9
        const fitsHeight = img.naturalHeight <= window.innerHeight * 0.9
        this.fullscreenCanZoom = !(fitsWidth && fitsHeight)
        this.fetchFullscreenImageInfo(img.src)
      }
    },
    openFullscreenGallery (index) {
      this.fullscreenIndex = index
      this.fullscreenImageLoaded = false
      this.fullscreenGallery = true
      document.body.style.overflow = 'hidden'
    },
    retryCarouselImage (index) {
      const retries = (this.carouselImageRetries[index] || 0) + 1
      this.carouselImageRetries[index] = retries
      const delay = retries * 1500 // 1.5s, 3s, 4.5s
      setTimeout(() => {
        this.carouselImageKeys[index] = (this.carouselImageKeys[index] || 0) + 1
      }, delay)
    },
    scrollToActiveIndicator (value) {
      const indicators = this.$el.querySelector?.('.carousel-indicator') || document.querySelector('.modal.is-active .carousel-indicator')
      const active = indicators?.children[value]
      if (!active) return
      indicators.scrollTo({
        top: 0,
        left: active.offsetLeft + active.offsetWidth / 2 - indicators.offsetWidth / 2,
        behavior: 'smooth'
      })
    },
    handleGalleryClick () {
      if (this.fullscreenZoomed) {
        this.fullscreenZoomed = false
      } else {
        this.closeFullscreenGallery()
      }
    }
  }
}
