import { type ReactNode, useEffect, useMemo, useRef, useState } from "react";
import { Minus, PanelLeft, SquareDashed, Trash2, X } from "lucide-react";

export type WindowDefinition = {
  id:string;
  title:string;
  icon?:ReactNode;
  content:ReactNode;
  initial:{x:number;y:number;width:number;height:number};
  activation?:number;
  kind?:"primary"|"context";
};
type WindowState = { x:number;y:number;width:number;height:number; minimized:boolean; closed:boolean; z:number; dock?:"left"|"right" };
type Drag = { id:string; mode:"move"|"resize"; startX:number;startY:number; initial:WindowState };

const storageKey="keelmesh.m6.window-layout.v1";
const reservedBottom=64;
function clamp(value:number,min:number,max:number){return Math.max(min,Math.min(max,value))}

export function WindowManager({windows}:{windows:WindowDefinition[]}){
  const defaults=useMemo(()=>Object.fromEntries(windows.map((w,i)=>[w.id,{...w.initial,minimized:false,closed:false,z:120+i}] as const)),[windows]);
  const [states,setStates]=useState<Record<string,WindowState>>(()=>{try{return {...defaults,...JSON.parse(localStorage.getItem(storageKey)??"{}")}}catch{return defaults}});
  const drag=useRef<Drag|null>(null);
  const activations=useRef<Record<string,number>>({});
  useEffect(()=>{setStates(current=>{let changed=false;const next={...current};for(const w of windows)if(!next[w.id]){next[w.id]=defaults[w.id];changed=true}return changed?next:current})},[windows,defaults]);
  useEffect(()=>{for(const w of windows){const token=w.activation??0;if(token>0&&token!==activations.current[w.id]){activations.current[w.id]=token;setStates(current=>{const state=current[w.id]??defaults[w.id];if(!state)return current;return{...current,[w.id]:{...state,z:Math.max(...Object.values(current).map(value=>value.z),120)+1,closed:false,minimized:false}}})}}},[windows,defaults]);
  useEffect(()=>{localStorage.setItem(storageKey,JSON.stringify(states))},[states]);
  useEffect(()=>{const move=(e:PointerEvent)=>{const d=drag.current;if(!d)return;setStates(s=>{const maxX=window.innerWidth-180,maxY=Math.max(82,window.innerHeight-reservedBottom-180);if(d.mode==="move")return{...s,[d.id]:{...s[d.id],x:clamp(d.initial.x+e.clientX-d.startX,0,maxX),y:clamp(d.initial.y+e.clientY-d.startY,44,maxY),dock:undefined}};return{...s,[d.id]:{...s[d.id],width:clamp(d.initial.width+e.clientX-d.startX,280,window.innerWidth-d.initial.x),height:clamp(d.initial.height+e.clientY-d.startY,180,window.innerHeight-d.initial.y-reservedBottom)}}})};const up=()=>{drag.current=null};window.addEventListener("pointermove",move);window.addEventListener("pointerup",up);return()=>{window.removeEventListener("pointermove",move);window.removeEventListener("pointerup",up)}},[]);
  const focus=(id:string)=>setStates(s=>{const current=s[id]??defaults[id];if(!current)return s;return{...s,[id]:{...current,z:Math.max(...Object.values(s).map(v=>v.z),120)+1,closed:false,minimized:false}}});
  const mutate=(id:string,patch:Partial<WindowState>)=>setStates(s=>({...s,[id]:{...(s[id]??defaults[id]),...patch}}));
  const minimized=windows.filter(w=>w.kind==="context"&&(states[w.id]??defaults[w.id])?.minimized&&!(states[w.id]??defaults[w.id])?.closed);
  return <>
    {windows.map(w=>{const s=states[w.id]??defaults[w.id];if(!s||s.closed||s.minimized)return null;const top=clamp(s.y,82,Math.max(82,window.innerHeight-reservedBottom-180));const style=s.dock?{left:s.dock==="left"?0:undefined,right:s.dock==="right"?0:undefined,top:82,width:Math.min(390,window.innerWidth*.34),height:window.innerHeight-82-reservedBottom,zIndex:s.z}:{left:s.x,top,width:s.width,height:Math.min(s.height,window.innerHeight-top-reservedBottom),zIndex:s.z};return <section key={w.id} className={`float-window ${s.dock?`docked ${s.dock}`:""}`} style={style} onPointerDown={()=>focus(w.id)} onKeyDown={e=>{if(!(e.altKey&&["ArrowLeft","ArrowRight","ArrowUp","ArrowDown"].includes(e.key)))return;e.preventDefault();const step=e.shiftKey?40:10;mutate(w.id,{dock:undefined,x:clamp(s.x+(e.key==="ArrowLeft"?-step:e.key==="ArrowRight"?step:0),0,window.innerWidth-180),y:clamp(s.y+(e.key==="ArrowUp"?-step:e.key==="ArrowDown"?step:0),82,window.innerHeight-reservedBottom-180)})}} tabIndex={0} aria-label={w.title}>
      <header className="float-title" onDoubleClick={()=>mutate(w.id,{dock:s.dock?undefined:"right"})} onPointerDown={e=>{if((e.target as HTMLElement).closest("button"))return;drag.current={id:w.id,mode:"move",startX:e.clientX,startY:e.clientY,initial:s};(e.currentTarget as HTMLElement).setPointerCapture?.(e.pointerId)}}>
        <span>{w.icon??<SquareDashed/>}</span><strong>{w.title}</strong><small>{s.dock?"DOCKED":"FLOATING"}</small>
        <button aria-label="Dock left" title="Dock left" onClick={()=>mutate(w.id,{dock:s.dock==="left"?undefined:"left"})}><PanelLeft/></button>
        <button aria-label="Minimize" title="Minimize" onClick={()=>mutate(w.id,{minimized:true})}><Minus/></button>
        {w.kind==="context"&&<button aria-label="Close" title="Close" onClick={()=>mutate(w.id,{closed:true,minimized:false})}><X/></button>}
      </header><div className="float-content">{w.content}</div><i className="resize-grip" onPointerDown={e=>{drag.current={id:w.id,mode:"resize",startX:e.clientX,startY:e.clientY,initial:s};e.currentTarget.setPointerCapture?.(e.pointerId)}} /></section>})}
    <div className="window-shelf" role="group" aria-label="Minimized detail windows"><span className="shelf-label">MINIMIZED</span>{minimized.length===0?<small>No detail windows</small>:minimized.map(w=><div className="task-item" key={w.id}><button className="task-restore" onClick={()=>focus(w.id)}><span>{w.icon??<SquareDashed/>}</span><b>{w.title}</b><Minus/></button><button className="task-trash" aria-label={`Close ${w.title}`} title={`Close ${w.title}`} onClick={()=>mutate(w.id,{closed:true,minimized:false})}><Trash2/></button></div>)}</div>
  </>;
}
