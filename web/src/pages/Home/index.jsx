/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React, { useContext, useEffect, useState } from 'react';
import {
  Button,
  Input,
  ScrollList,
  ScrollItem,
  Tabs,
  TabPane,
} from '@douyinfe/semi-ui';
import { API, showError, copy, showSuccess } from '../../helpers';
import { useIsMobile } from '../../hooks/common/useIsMobile';
import { API_ENDPOINTS } from '../../constants/common.constant';
import { StatusContext } from '../../context/Status';
import { useActualTheme } from '../../context/Theme';
import { marked } from 'marked';
import { useTranslation } from 'react-i18next';
import {
  IconGithubLogo,
  IconPlay,
  IconCopy,
} from '@douyinfe/semi-icons';
import { Link } from 'react-router-dom';
import NoticeModal from '../../components/layout/NoticeModal';
import ModelAvailabilityEntryCard from '../../components/home/ModelAvailabilityEntryCard';

const Home = () => {
  const { t, i18n } = useTranslation();
  const [statusState] = useContext(StatusContext);
  const actualTheme = useActualTheme();
  const [homePageContentLoaded, setHomePageContentLoaded] = useState(false);
  const [homePageContent, setHomePageContent] = useState('');
  const [noticeVisible, setNoticeVisible] = useState(false);
  const isMobile = useIsMobile();
  const isDemoSiteMode = statusState?.status?.demo_site_enabled || false;
  const serverAddress =
    statusState?.status?.server_address || `${window.location.origin}`;
  const endpointItems = API_ENDPOINTS.map((e) => ({ value: e }));
  const [endpointIndex, setEndpointIndex] = useState(0);
  const [activePlatform, setActivePlatform] = useState('windows');
  const currentLanguage = i18n.language || 'zh-CN';
  const isChinese = currentLanguage.startsWith('zh');

  const displayHomePageContent = async () => {
    try {
      setHomePageContent(localStorage.getItem('home_page_content') || '');
      const res = await API.get('/api/home_page_content');
      const { success, message, data } = res.data;
      if (success) {
        const raw = (data || '').trim();
        let content = raw;
        if (raw && !raw.startsWith('https://')) {
          content = marked.parse(raw);
        }
        setHomePageContent(content);
        localStorage.setItem('home_page_content', content);

        // 如果内容是 URL，则发送主题模式
        if (raw.startsWith('https://')) {
          const iframe = document.querySelector('iframe');
          if (iframe) {
            iframe.onload = () => {
              iframe.contentWindow.postMessage({ themeMode: actualTheme }, '*');
              iframe.contentWindow.postMessage({ lang: currentLanguage }, '*');
            };
          }
        }
      } else {
        showError(message);
        setHomePageContent('');
      }
    } catch (error) {
      console.error('加载首页内容失败:', error);
      setHomePageContent('');
      showError(t('加载首页内容失败...'));
    } finally {
      setHomePageContentLoaded(true);
    }
  };

  const handleCopyBaseURL = async () => {
    const ok = await copy(serverAddress);
    if (ok) {
      showSuccess(t('已复制到剪切板'));
    }
  };

  useEffect(() => {
    const checkNoticeAndShow = async () => {
      const lastCloseDate = localStorage.getItem('notice_close_date');
      const today = new Date().toDateString();
      if (lastCloseDate !== today) {
        try {
          const res = await API.get('/api/notice');
          const { success, data } = res.data;
          if (success && data && data.trim() !== '') {
            setNoticeVisible(true);
          }
        } catch (error) {
          console.error('获取公告失败:', error);
        }
      }
    };

    checkNoticeAndShow();
  }, []);

  useEffect(() => {
    displayHomePageContent().then();
  }, []);

  useEffect(() => {
    const timer = setInterval(() => {
      setEndpointIndex((prev) => (prev + 1) % endpointItems.length);
    }, 3000);
    return () => clearInterval(timer);
  }, [endpointItems.length]);

  const currentEndpoint = endpointItems[endpointIndex]?.value || '/v1/chat/completions';
  const fullEndpoint = `${serverAddress}${currentEndpoint}`;

  const handleCopyText = async (text, successText = t('已复制到剪切板')) => {
    const ok = await copy(text);
    if (ok) showSuccess(successText);
  };

  const scrollToExamples = () => {
    const el = document.getElementById('integration-examples');
    if (el) {
      el.scrollIntoView({ behavior: 'smooth', block: 'start' });
    }
  };

  const integrationExamples = [
    {
      key: 'claude-code',
      title: 'Claude Code',
      description: '已安装 Claude Code 后，复制下面命令，替换令牌后直接运行。',
      platforms: {
        windows: {
          label: 'Windows PowerShell',
          command: `$env:ANTHROPIC_BASE_URL = "${serverAddress}"
$env:ANTHROPIC_API_KEY = "sk-你的令牌"

claude code --model claude-3-7-sonnet-latest`,
        },
        macos: {
          label: 'macOS Terminal',
          command: `export ANTHROPIC_BASE_URL="${serverAddress}"
export ANTHROPIC_API_KEY="sk-你的令牌"

claude code --model claude-3-7-sonnet-latest`,
        },
        linux: {
          label: 'Linux Terminal',
          command: `export ANTHROPIC_BASE_URL="${serverAddress}"
export ANTHROPIC_API_KEY="sk-你的令牌"

claude code --model claude-3-7-sonnet-latest`,
        },
      },
    },
    {
      key: 'codex',
      title: 'Codex',
      description: '已安装 Codex CLI 后，复制下面命令，替换令牌后直接运行。',
      platforms: {
        windows: {
          label: 'Windows PowerShell',
          command: `$env:OPENAI_BASE_URL = "${serverAddress}/v1"
$env:OPENAI_API_KEY = "sk-你的令牌"

codex`,
        },
        macos: {
          label: 'macOS Terminal',
          command: `export OPENAI_BASE_URL="${serverAddress}/v1"
export OPENAI_API_KEY="sk-你的令牌"

codex`,
        },
        linux: {
          label: 'Linux Terminal',
          command: `export OPENAI_BASE_URL="${serverAddress}/v1"
export OPENAI_API_KEY="sk-你的令牌"

codex`,
        },
      },
    },
    {
      key: 'openclaw',
      title: 'OpenClaw',
      description: '已安装 OpenClaw 后，复制下面命令，替换令牌后直接运行。',
      platforms: {
        windows: {
          label: 'Windows PowerShell',
          command: `$env:OPENAI_BASE_URL = "${serverAddress}/v1"
$env:OPENAI_API_KEY = "sk-你的令牌"

openclaw`,
        },
        macos: {
          label: 'macOS Terminal',
          command: `export OPENAI_BASE_URL="${serverAddress}/v1"
export OPENAI_API_KEY="sk-你的令牌"

openclaw`,
        },
        linux: {
          label: 'Linux Terminal',
          command: `export OPENAI_BASE_URL="${serverAddress}/v1"
export OPENAI_API_KEY="sk-你的令牌"

openclaw`,
        },
      },
    },
  ];

  const platformMeta = {
    windows: {
      label: 'Windows',
      shell: 'PowerShell',
    },
    macos: {
      label: 'macOS',
      shell: 'Terminal',
    },
    linux: {
      label: 'Linux',
      shell: 'Terminal',
    },
  };

  return (
    <div className='w-full overflow-x-hidden'>
      <NoticeModal
        visible={noticeVisible}
        onClose={() => setNoticeVisible(false)}
        isMobile={isMobile}
      />
      {homePageContentLoaded && homePageContent.trim() === '' ? (
        <div className='w-full overflow-x-hidden'>
          <div className='relative border-b border-semi-color-border'>
            <div
              className='absolute inset-0 pointer-events-none opacity-40'
              style={{
                backgroundImage:
                  'linear-gradient(to right, rgba(99,102,241,0.1) 1px, transparent 1px), linear-gradient(to bottom, rgba(99,102,241,0.1) 1px, transparent 1px)',
                backgroundSize: '40px 40px',
              }}
            />
            <div className='blur-ball blur-ball-indigo' />
            <div className='blur-ball blur-ball-teal' />

            <div className='relative max-w-6xl mx-auto px-4 pt-28 pb-16 md:pt-32 md:pb-20'>
              <div className='grid grid-cols-1 lg:grid-cols-2 gap-8 lg:gap-10 items-center'>
                <div>
                  <div className='inline-flex items-center px-3 py-1 rounded-full border border-semi-color-border bg-semi-color-bg-1 text-sm mb-5 shadow-sm'>
                    API Plus · AI Gateway
                  </div>
                  <h1
                    className={`text-4xl md:text-5xl xl:text-6xl font-bold leading-tight text-semi-color-text-0 ${isChinese ? 'tracking-wide md:tracking-wider' : ''}`}
                  >
                    {t('统一的')}
                    <br />
                    <span className='shine-text'>{t('大模型接口网关')}</span>
                  </h1>
                  <p className='mt-5 text-base md:text-lg text-semi-color-text-1 max-w-xl'>
                    {t('更好的价格，更好的稳定性，只需要将模型基址替换为：')}
                  </p>

                  <div className='mt-6 flex flex-col gap-3 max-w-xl'>
                    <Input
                      readonly
                      value={serverAddress}
                      className='!rounded-2xl'
                      size={isMobile ? 'default' : 'large'}
                      suffix={
                        <Button
                          type='primary'
                          theme='solid'
                          onClick={handleCopyBaseURL}
                          icon={<IconCopy />}
                          className='!rounded-xl'
                        />
                      }
                    />
                    <div className='flex items-center gap-3 p-2 rounded-2xl border border-semi-color-border bg-semi-color-bg-1'>
                      <ScrollList
                        bodyHeight={32}
                        style={{
                          border: 'unset',
                          boxShadow: 'unset',
                          width: '100%',
                        }}
                      >
                        <ScrollItem
                          mode='wheel'
                          cycled={true}
                          list={endpointItems}
                          selectedIndex={endpointIndex}
                          onSelect={({ index }) => setEndpointIndex(index)}
                        />
                      </ScrollList>
                      <Button
                        icon={<IconCopy />}
                        className='!rounded-xl'
                        onClick={() => copy(fullEndpoint)}
                      >
                        Full
                      </Button>
                    </div>
                  </div>

                  <div className='mt-7 flex flex-wrap gap-3'>
                    <Link to='/console'>
                      <Button
                        theme='solid'
                        type='primary'
                        size={isMobile ? 'default' : 'large'}
                        className='!rounded-2xl px-8'
                        icon={<IconPlay />}
                      >
                        {t('获取密钥')}
                      </Button>
                    </Link>
                    <Button
                      size={isMobile ? 'default' : 'large'}
                      className='!rounded-2xl px-8'
                      onClick={scrollToExamples}
                    >
                      接入示例
                    </Button>
                    <Button
                      size={isMobile ? 'default' : 'large'}
                      className='!rounded-2xl px-8'
                      onClick={scrollToExamples}
                    >
                      文档
                    </Button>
                    {isDemoSiteMode && statusState?.status?.version ? (
                      <Button
                        size={isMobile ? 'default' : 'large'}
                        className='!rounded-2xl px-6'
                        icon={<IconGithubLogo />}
                        onClick={() => window.open('https://api.apiplus.cloud', '_blank')}
                      >
                        {statusState.status.version}
                      </Button>
                    ) : null}
                  </div>
                </div>

                <div className='rounded-3xl border border-semi-color-border bg-semi-color-bg-1 p-6 md:p-8 shadow-lg'>
                  <div className='flex items-center justify-between mb-4'>
                    <div className='text-semi-color-text-0 text-xl font-semibold'>
                      Terminal Preview
                    </div>
                    <span className='text-xs text-semi-color-text-2'>LIVE</span>
                  </div>
                  <div className='rounded-2xl bg-[#0B1220] text-[#94A3B8] p-4 font-mono text-xs md:text-sm leading-6 overflow-x-auto'>
                    <div className='text-[#22D3EE]'>$ export API_BASE="{serverAddress}"</div>
                    <div>$ export API_KEY="sk-***"</div>
                    <div className='text-[#A78BFA]'>$ curl {fullEndpoint}</div>
                    <div>{'{'}</div>
                    <div className='pl-4'>"id": "chatcmpl-xxx",</div>
                    <div className='pl-4'>"model": "gpt-4o-mini",</div>
                    <div className='pl-4'>"choices": [...]</div>
                    <div>{'}'}</div>
                  </div>
                  <div className='mt-4 rounded-2xl bg-semi-color-fill-0 p-4'>
                    <div className='text-xs text-semi-color-text-2 mb-1'>当前完整请求地址</div>
                    <div className='font-mono text-sm break-all'>{fullEndpoint}</div>
                  </div>
                </div>
              </div>

              <div className='mt-14'>
                <div className='grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-4'>
                  <div className='rounded-2xl border border-semi-color-border p-5 bg-semi-color-bg-0'>
                    <div className='text-lg font-semibold'>统一接入</div>
                    <div className='text-semi-color-text-1 mt-2 text-sm'>
                      支持 OpenAI / Claude / Gemini 等格式，减少多平台对接成本。
                    </div>
                  </div>
                  <div className='rounded-2xl border border-semi-color-border p-5 bg-semi-color-bg-0'>
                    <div className='text-lg font-semibold'>更稳更省</div>
                    <div className='text-semi-color-text-1 mt-2 text-sm'>
                      渠道管理、模型路由与重试机制结合，让调用更稳定、成本更可控。
                    </div>
                  </div>
                  <div className='rounded-2xl border border-semi-color-border p-5 bg-semi-color-bg-0'>
                    <div className='text-lg font-semibold'>开箱可用</div>
                    <div className='text-semi-color-text-1 mt-2 text-sm'>
                      注册即可生成密钥，替换 Base URL 后可快速开始请求。
                    </div>
                  </div>
                  <ModelAvailabilityEntryCard />
                </div>
              </div>

              <div id='integration-examples' className='mt-14'>
                <div className='mb-5'>
                  <h2 className='text-2xl md:text-3xl font-bold'>
                    Claude Code / Codex / OpenClaw 接入示例
                  </h2>
                  <p className='text-semi-color-text-1 mt-2 text-sm md:text-base'>
                    先选择你的系统，再复制完整命令；把 <code>sk-你的令牌</code> 换成你在控制台创建的真实密钥即可。
                  </p>
                </div>
                <div className='mb-5 rounded-2xl border border-semi-color-border bg-semi-color-bg-1 p-3 md:p-4'>
                  <Tabs type='button' activeKey={activePlatform} onChange={setActivePlatform}>
                    <TabPane tab='Windows' itemKey='windows' />
                    <TabPane tab='macOS' itemKey='macos' />
                    <TabPane tab='Linux' itemKey='linux' />
                  </Tabs>
                  <div className='mt-3 text-xs md:text-sm text-semi-color-text-2'>
                    当前显示：{platformMeta[activePlatform].label} · {platformMeta[activePlatform].shell}
                  </div>
                </div>
                <div className='grid grid-cols-1 md:grid-cols-3 gap-4'>
                  {integrationExamples.map((item) => (
                    <div
                      key={item.key}
                      className='rounded-2xl border border-semi-color-border bg-semi-color-bg-1 p-4 md:p-5 shadow-sm'
                    >
                      <div className='flex items-center justify-between'>
                        <div className='text-lg font-semibold'>{item.title}</div>
                        <Button
                          icon={<IconCopy />}
                          size='small'
                          className='!rounded-xl'
                          onClick={() =>
                            handleCopyText(
                              item.platforms[activePlatform].command,
                              `${item.title} ${platformMeta[activePlatform].label} 示例已复制`,
                            )
                          }
                        />
                      </div>
                      <div className='text-semi-color-text-1 text-sm mt-2 mb-3'>
                        {item.description}
                      </div>
                      <div className='rounded-xl bg-semi-color-fill-0 p-3 text-sm mb-3'>
                        <div className='font-medium mb-2'>小白使用步骤</div>
                        <ol className='list-decimal pl-5 space-y-1 text-semi-color-text-1'>
                          <li>点击上方“获取密钥”，先创建一个 API Key。</li>
                          <li>复制下面整段命令，粘贴到 {platformMeta[activePlatform].shell}。</li>
                          <li>把命令里的 <code>sk-你的令牌</code> 改成你的真实密钥后回车运行。</li>
                        </ol>
                      </div>
                      <div className='text-xs text-semi-color-text-2 mb-2'>
                        适用环境：{item.platforms[activePlatform].label}
                      </div>
                      <pre className='rounded-xl bg-[#0B1220] text-[#E2E8F0] p-3 text-xs leading-5 overflow-x-auto whitespace-pre-wrap break-all'>
                        <code>{item.platforms[activePlatform].command}</code>
                      </pre>
                    </div>
                  ))}
                </div>
                <div className='mt-4 text-xs text-semi-color-text-2'>
                  提示：Codex / OpenClaw 示例使用 OpenAI 兼容地址（{serverAddress}/v1）。
                </div>
              </div>

              <div className='mt-8'>
                <div className='rounded-2xl border border-semi-color-border bg-semi-color-bg-1 p-4 flex flex-wrap items-center justify-between gap-3'>
                  <div className='text-sm text-semi-color-text-1'>
                    文档按钮已跳转到接入示例区，你也可以直接复制上面的命令快速接入。
                  </div>
                  <div className='flex gap-2'>
                    <Button className='!rounded-xl' onClick={scrollToExamples}>
                      接入示例
                    </Button>
                    <Button
                      icon={<IconCopy />}
                      className='!rounded-xl'
                      onClick={() => handleCopyText(serverAddress)}
                    >
                      复制 Base URL
                    </Button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      ) : (
        <div className='overflow-x-hidden w-full'>
          {homePageContent.trim().startsWith('https://') ? (
            <iframe
              src={homePageContent}
              className='w-full h-screen border-none'
            />
          ) : (
            <div
              className='mt-[60px]'
              dangerouslySetInnerHTML={{ __html: homePageContent }}
            />
          )}
        </div>
      )}
    </div>
  );
};

export default Home;
