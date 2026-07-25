import schedMindAppIcon from '../assets/schedmind-app-icon.png'
import {
  applicationRootPath,
} from './navigation'

export function TopBar() {
  return (
    <header className="border-b border-[#EBF0F5] bg-white px-5 py-4 sm:px-8 lg:px-12">
      <div className="mx-auto flex h-9 max-w-7xl items-center pl-14">
        <a
          href={applicationRootPath}
          className="flex items-center gap-3 rounded-xl focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#0C4DA2]"
          aria-label="SchedMind home"
          onClick={(event) => {
            if (
              event.button !== 0 ||
              event.metaKey ||
              event.ctrlKey ||
              event.shiftKey ||
              event.altKey
            ) {
              return
            }
            event.preventDefault()
            window.history.pushState(null, '', applicationRootPath)
            window.dispatchEvent(new PopStateEvent('popstate'))
          }}
        >
        <span
          className="grid size-9 shrink-0 place-items-center overflow-hidden rounded-xl"
          aria-hidden="true"
        >
          <img
            className="size-full scale-[1.3]"
            src={schedMindAppIcon}
            alt=""
          />
        </span>
        <span className="font-black">SchedMind</span>
        </a>
      </div>
    </header>
  )
}
