"use client";

import { Menu, PanelLeftClose } from "lucide-react";

import { SidebarContent } from "@/components/sentinel/dashboard";
import { Button } from "@/components/ui/button";
import {
  Drawer,
  DrawerClose,
  DrawerContent,
  DrawerDescription,
  DrawerTitle,
  DrawerTrigger,
} from "@/components/ui/drawer";
import { cn } from "@/lib/utils";

export default function SidebarDrawer({
  className,
}: {
  className?: string;
}) {
  return (
    <Drawer swipeDirection="left">
      <DrawerTrigger
        render={
          <Button
            variant="outline"
            size="icon"
            aria-label="Open navigation"
            className={cn(
              "cursor-pointer border-slate-800 bg-slate-950/70 text-slate-300 shadow-[0_12px_34px_rgba(0,0,0,0.16)] hover:border-slate-700 hover:bg-slate-900 hover:text-white",
              className,
            )}
          />
        }
      >
        <Menu className="size-4" />
      </DrawerTrigger>

      <DrawerContent className="border-slate-800 bg-[#090d14]/98 [--drawer-content-width:17.5rem] sm:[--drawer-content-width:18.5rem]">
        <DrawerTitle className="sr-only">Sentinel navigation</DrawerTitle>
        <DrawerDescription className="sr-only">
          Primary navigation for the Sentinel analyst console.
        </DrawerDescription>

        <div className="relative h-dvh px-5 py-6">
          <DrawerClose
            render={
              <Button
                variant="outline"
                size="icon"
                aria-label="Close navigation"
                className="absolute right-4 top-4 z-20 cursor-pointer border-slate-800 bg-slate-950/75 text-slate-500 hover:bg-slate-900 hover:text-slate-100"
              />
            }
          >
            <PanelLeftClose className="size-4" />
          </DrawerClose>

          <SidebarContent />
        </div>
      </DrawerContent>
    </Drawer>
  );
}
