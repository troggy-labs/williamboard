/** @type {import('next').NextConfig} */
const nextConfig = {
  async rewrites() {
    return [
      {
        source: '/api/:path*',
        destination: 'http://localhost:8080/v1/:path*', // Proxy to backend
      },
    ]
  },
  // Increase timeout for long-running requests
  experimental: {
    proxyTimeout: 120000, // 2 minutes
  },
}

module.exports = nextConfig