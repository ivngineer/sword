import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MainScreen } from "../screens/MainScreen";
import "../index.css";

document.documentElement.setAttribute("data-theme", "dark");
document.documentElement.className = "dark";

const queryClient = new QueryClient();

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <MainScreen />
    </QueryClientProvider>
  </React.StrictMode>,
);
