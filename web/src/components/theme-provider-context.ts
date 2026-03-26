import * as React from "react"

type Theme = "dark" | "light" | "system"

type ThemeProviderState = {
  theme: Theme
  setTheme: (theme: Theme) => void
}

const ThemeProviderContext = React.createContext<ThemeProviderState>({
  theme: "system",
  setTheme: () => undefined,
})

function useTheme() {
  return React.useContext(ThemeProviderContext)
}

export { ThemeProviderContext, useTheme, type Theme, type ThemeProviderState }
