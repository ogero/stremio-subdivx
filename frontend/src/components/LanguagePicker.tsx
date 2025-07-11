"use client"

import {DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger} from "@/components/ui/dropdown-menu"
import {Button} from "@/components/ui/button"
import React from "react"
import {useTranslation} from "react-i18next";

export default function LanguagePicker() {

  const {i18n} = useTranslation();

  const toggleLanguage = () => {
    void i18n.changeLanguage(i18n.language == 'en' ? 'es' : 'en');
  };

  return (
    <React.Fragment>
      <div className="hidden sm:block">
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="outline" className="flex items-center gap-2">
              <GlobeIcon className="h-5 w-5"/>
              <span>{i18n.language.toUpperCase()}</span>
              <ChevronDownIcon className="h-4 w-4"/>
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-48">
            <DropdownMenuItem className="flex items-center justify-between" onSelect={() => {
              toggleLanguage()
            }}>
              <span>{i18n.language == 'en' ? 'English' : 'Español'}</span>
              <CheckIcon className="h-5 w-5"/>
            </DropdownMenuItem>
            <DropdownMenuItem onSelect={() => {
              toggleLanguage()
            }}>{i18n.language == 'en' ? 'Español' : 'English'}</DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </React.Fragment>
  )
}

function CheckIcon(props) {
  return (
    <svg
      {...props}
      xmlns="http://www.w3.org/2000/svg"
      width="24"
      height="24"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <path d="M20 6 9 17l-5-5"/>
    </svg>
  )
}


function ChevronDownIcon(props) {
  return (
    <svg
      {...props}
      xmlns="http://www.w3.org/2000/svg"
      width="24"
      height="24"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <path d="m6 9 6 6 6-6"/>
    </svg>
  )
}


function GlobeIcon(props) {
  return (
    <svg
      {...props}
      xmlns="http://www.w3.org/2000/svg"
      width="24"
      height="24"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <circle cx="12" cy="12" r="10"/>
      <path d="M12 2a14.5 14.5 0 0 0 0 20 14.5 14.5 0 0 0 0-20"/>
      <path d="M2 12h20"/>
    </svg>
  )
}

