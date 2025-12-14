import Link from 'next/link';

const features = [
  {
    icon: '🔐',
    title: 'ECH 加密',
    description: '基于 Encrypted Client Hello 技术，隐藏真实访问目标',
  },
  {
    icon: '🖥️',
    title: '跨平台支持',
    description: '支持 Windows、macOS、Linux 桌面客户端',
  },
  {
    icon: '💻',
    title: '命令行工具',
    description: '轻量级 CLI 客户端，适合服务器和自动化场景',
  },
  {
    icon: '🌐',
    title: '多协议支持',
    description: '同时支持 SOCKS5 和 HTTP 代理协议',
  },
  {
    icon: '🚦',
    title: '智能分流',
    description: '支持全局代理、跳过中国大陆、直连等多种模式',
  },
  {
    icon: '⚡',
    title: '高性能',
    description: '基于 Go 语言开发，低资源占用，高并发处理',
  },
];

const quickLinks = [
  {
    title: '技术原理',
    description: '了解 ECH 技术和 EchPlus 工作原理',
    href: '/docs/principle',
  },
  {
    title: '服务端部署',
    description: '在服务器上部署 EchPlus 服务端',
    href: '/docs/server',
  },
  {
    title: '命令行客户端',
    description: '使用 CLI 客户端连接代理',
    href: '/docs/client',
  },
  {
    title: '桌面端安装',
    description: '下载安装图形化桌面客户端',
    href: '/docs/desktop',
  },
];

export default function HomePage() {
  return (
    <div className="flex flex-col items-center">
      {/* Hero Section */}
      <section className="flex flex-col items-center justify-center text-center py-20 px-4 max-w-4xl">
        <h1 className="text-4xl md:text-5xl font-bold mb-6 bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent">
          EchPlus
        </h1>
        <p className="text-xl text-fd-muted-foreground mb-8 max-w-2xl">
          基于 ECH (Encrypted Client Hello) 技术的安全代理工具，
          保护您的网络隐私
        </p>
        <div className="flex gap-4 flex-wrap justify-center">
          <Link
            href="/docs"
            className="px-6 py-3 bg-fd-primary text-fd-primary-foreground rounded-lg font-medium hover:opacity-90 transition-opacity"
          >
            开始使用
          </Link>
          <Link
            href="https://github.com/atticus6/echPlus"
            className="px-6 py-3 border border-fd-border rounded-lg font-medium hover:bg-fd-accent transition-colors"
            target="_blank"
          >
            GitHub
          </Link>
        </div>
      </section>

      {/* Features Section */}
      <section className="w-full max-w-6xl px-4 py-16">
        <h2 className="text-2xl font-bold text-center mb-12">核心特性</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {features.map((feature) => (
            <div
              key={feature.title}
              className="p-6 rounded-xl border border-fd-border bg-fd-card hover:shadow-lg transition-shadow"
            >
              <div className="text-3xl mb-4">{feature.icon}</div>
              <h3 className="text-lg font-semibold mb-2">{feature.title}</h3>
              <p className="text-fd-muted-foreground text-sm">
                {feature.description}
              </p>
            </div>
          ))}
        </div>
      </section>

      {/* Quick Start Section */}
      <section className="w-full max-w-6xl px-4 py-16">
        <h2 className="text-2xl font-bold text-center mb-12">快速开始</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          {quickLinks.map((link) => (
            <Link
              key={link.title}
              href={link.href}
              className="p-6 rounded-xl border border-fd-border bg-fd-card hover:border-fd-primary transition-colors group"
            >
              <h3 className="text-lg font-semibold mb-2 group-hover:text-fd-primary transition-colors">
                {link.title} →
              </h3>
              <p className="text-fd-muted-foreground text-sm">
                {link.description}
              </p>
            </Link>
          ))}
        </div>
      </section>

      {/* Architecture Section */}
      <section className="w-full max-w-4xl px-4 py-16">
        <h2 className="text-2xl font-bold text-center mb-8">架构概览</h2>
        <div className="p-6 rounded-xl border border-fd-border bg-fd-card">
          <pre className="text-sm overflow-x-auto text-center">
{`┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   应用程序   │────▶│   EchPlus   │────▶│   服务端    │────▶ 目标网站
│  (浏览器等)  │     │    客户端   │     │ (WebSocket) │
└─────────────┘     └─────────────┘     └─────────────┘
    SOCKS5/HTTP         ECH + WSS           TCP`}
          </pre>
        </div>
      </section>

      {/* Footer */}
      <footer className="w-full border-t border-fd-border py-8 mt-8">
        <div className="max-w-6xl mx-auto px-4 text-center text-fd-muted-foreground text-sm">
          <p>基于 MIT License 开源</p>
          <p className="mt-2">
            <Link
              href="https://github.com/atticus6/echPlus"
              className="hover:text-fd-foreground transition-colors"
              target="_blank"
            >
              GitHub
            </Link>
            {' · '}
            <Link
              href="/docs"
              className="hover:text-fd-foreground transition-colors"
            >
              文档
            </Link>
          </p>
        </div>
      </footer>
    </div>
  );
}
