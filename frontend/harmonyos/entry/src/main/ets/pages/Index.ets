import { webview } from '@kit.ArkWeb';
import { hilog } from '@kit.PerformanceAnalysisKit';

const DOMAIN = 0x0000;
const WXX_URL: string = 'https://wxx-agent.online';

@Entry
@Component
struct Index {
  controller: webview.WebviewController = new webview.WebviewController();
  @State currentUrl: string = WXX_URL;
  @State loadFailed: boolean = false;

  onPageShow(): void {
    hilog.info(DOMAIN, 'wxx', 'WebView page shown');
  }

  build() {
    Column() {
      if (this.loadFailed) {
        Column({ space: 16 }) {
          Text('页面加载失败')
            .fontSize(20)
            .fontColor('#333333')
          Text('请检查网络后点击重试')
            .fontSize(14)
            .fontColor('#999999')
          Button('重新加载')
            .onClick(() => {
              this.loadFailed = false;
              this.controller.loadUrl(WXX_URL);
            })
        }
        .width('100%')
        .layoutWeight(1)
        .justifyContent(FlexAlign.Center)
      } else {
        Web({ src: this.currentUrl, controller: this.controller })
          .width('100%')
          .height('100%')
          .javaScriptAccess(true)
          .domStorageAccess(true)
          .geolocationAccess(true)
          .onErrorReceive((event) => {
            hilog.error(DOMAIN, 'wxx', 'WebView onErrorReceive: %{public}s', JSON.stringify(event));
          })
          .onHttpErrorReceive((event) => {
            const code: number = event.response.getResponseCode();
            hilog.warn(DOMAIN, 'wxx', 'WebView onHttpErrorReceive: %{public}d', code);
            if (code === 404 || code >= 500) {
              this.loadFailed = true;
            }
          })
      }
    }
    .width('100%')
    .height('100%')
  }
}
