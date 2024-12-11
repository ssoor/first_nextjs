"use client";

import { MagnifyingGlassIcon } from "@heroicons/react/24/outline";
import { useSearchParams, useRouter, usePathname } from "next/navigation";
import { useDebouncedCallback } from "use-debounce";


import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectTrigger,
  SelectValue,
} from "@/app/ui/select"

export default function Search({ placeholder }: { placeholder: string }) {
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const { replace } = useRouter();

  const query = searchParams.get("query")?.toString();
  const status = searchParams.get('status') || undefined;

  const handleSearch = useDebouncedCallback((term: string) => {
    const params = new URLSearchParams(searchParams);
    if (term) {
      params.set("query", term);
    } else {
      params.delete("query");
    }
    console.log(term);
    params.set("page", "1");
    replace(`${pathname}?${params.toString()}`);
  }, 1000);

  const handleStatus = function (status: string) {
    const params = new URLSearchParams(searchParams);
    if (status) {
      params.set("status", status);
    } else {
      params.delete("status");
    }
    console.log(status);
    params.set("page", "1");
    replace(`${pathname}?${params.toString()}`);
  };

  return (
    <>
      <div className="relative flex flex-1 flex-shrink-0">
        <label htmlFor="search" className="sr-only">
          Search
        </label>
        <input
          onChange={(e) => {
            handleSearch(e.target.value);
          }}
          defaultValue={query}
          className="peer block w-full rounded-md border border-gray-200 pl-10 text-sm outline-2 placeholder:text-gray-500"
          placeholder={placeholder}
        />
        <MagnifyingGlassIcon className="absolute left-3 top-1/2 h-[18px] w-[18px] -translate-y-1/2 text-gray-500 peer-focus:text-gray-900" />
      </div>

      <div>
        <Select value={status} onValueChange={(v) => { handleStatus(v); }}>
          <SelectTrigger className="w-[180px]">
            <SelectValue placeholder="全部状态" />
          </SelectTrigger>
          <SelectContent>
            <SelectGroup>
              <SelectItem value="1">即将开始</SelectItem>
              <SelectItem value="2">正在进行</SelectItem>
            </SelectGroup>
          </SelectContent>
        </Select>
      </div>
    </>
  );
}
