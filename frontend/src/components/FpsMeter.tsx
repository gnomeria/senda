// Lightweight FPS overlay. Measures requestAnimationFrame cadence — i.e. how
// often the main thread gets a frame callback, vsync-paced. NOTE: this is NOT
// the same as compositor paint rate. On WebKitGTK a scroll can drop to ~30fps
// visually while rAF still fires at 60 (paint/composite runs off-thread). So:
//   rAF ~60 but scroll looks janky  => bottleneck is paint/composite, not JS.
// For true GPU frame rate use MangoHud on the binary.
import { onCleanup, onMount } from "solid-js";

export default function FpsMeter() {
  let raf = 0;
  let last = performance.now();
  let frames = 0;
  let acc = 0;
  let fps = 0;
  let worst = 0; // worst (longest) frame interval in the window, ms
  let el!: HTMLDivElement;

  const tick = (now: number) => {
    const dt = now - last;
    last = now;
    frames++;
    acc += dt;
    if (dt > worst) worst = dt;
    if (acc >= 500) {
      fps = Math.round((frames * 1000) / acc);
      const lowFps = worst > 0 ? Math.round(1000 / worst) : fps;
      if (el) {
        el.textContent = `${fps} fps · ${(acc / frames).toFixed(1)} ms avg · low ${lowFps}`;
        el.classList.toggle("bad", fps < 50);
      }
      frames = 0;
      acc = 0;
      worst = 0;
    }
    raf = requestAnimationFrame(tick);
  };

  onMount(() => {
    raf = requestAnimationFrame(tick);
  });
  onCleanup(() => cancelAnimationFrame(raf));

  return <div class="fps-meter" ref={el}>– fps</div>;
}
