require ("gosvrk_ini")
require ("drycont")

--json = require "json"

--local outmsg[1]={["kind"]=0,["inputid"]=0,["msg"]=""}

local data={}
local modsdiag={}     --[inputid]["testl"..."workl"...]
local diagchg={}
local crlive={}   --live crates {["inputid_cr"]=packcnt,["inputid_cr"]=packcnt,...}
--local crself={}
local packcnt=0
local crtm={}
local crtms={}

local outmsg={}  --[0]={kind="",iid=0,msg=""}


function AddMsg(kind,iid,msg)
if #outmsg>10000 then 
	return 
end
--print("addmsg",#outmsg)

outmsg[#outmsg+1]={kind=kind,iid=iid,msg=msg}

--print("addmsg",outmsg[#outmsg].kind)
--print("addmsg")
end

--- from GO ---
function GetMsg(n)
if n>#outmsg then 
	outmsg={} 
	return nil,nil,nil 
end
--print ("getmsg",  kind: -3 in stack  
return outmsg[n].kind,outmsg[n].iid,outmsg[n].msg
end

--moddiag[0]=

--- from GO ---
function TestFun(v1,v2)
	print("LUA!!!",v1,v2)
	return "hoho",25
end


function AddDt(key,val)
		data[key]=val
end

--- from GO ---
function GetDt(key)
		return data[key]
end

-- OLD not use
function Add_ModDiag(inputid,testl,errl,modgood)

--modsdiag[inputid]={}
--modsdiag[inputid]["testl"]=testl
--modsdiag[inputid]["errl"]=errl
--modsdiag[inputid]["modgood"]=modgood

s=tostring(inputid)
modsdiag[s]={}
modsdiag[s]["testl"]=testl
modsdiag[s]["errl"]=errl
modsdiag[s]["modgood"]=modgood
--print ("testl",inputid,modsdiag[inputid]["testl"])
end


--- from GO ---
-- start of packet
function Add_UNOHead()
	--print ("Add_UNOHead")
	packcnt=packcnt+1
end

--- from GO ---
function End_Pack()
	--print("End Pack")
	
	--for crid,cnt in ipairs(crlive) do
	for crid,cnt in pairs(crlive) do
      	  
	   --print ("crates",crid,crlive[crid])
	   --if(packcnt-crlive[crid]>1) then
			--print("crate miss",crid)
			--crlive[crid]=nil
			--crself[crid]=0
       if (cnt~=packcnt) then
				--print ("crate miss in pack", crid)
				if(cnt~=0) then
					if(MsgMissCr==1) then
						AddMsg("errorpk",crid,"crate miss in pack: "..tostring(packcnt))
					end
				crlive[crid]=0
				end
	   end
  
    end
	
	--print("count:",collectgarbage())
end

--- from GO ---
function Add_CrateHead(inputid,basetime_lnxms,basetime_str)
	si=tostring(inputid)
	
	--if (crself[si]==nil) then 
	--	crself[si]=0
    --end
	
	--crself[si]=crself[si]+1
	
	--if(crself[si]<5) then return end

	if(crlive[si]==nil) then 
	  --print("Add_CrateHead",si,"crate add to table")
	  AddMsg("info",si,"new crate")
	end
	
	if(crlive[si]==0) then 
	  --print("Add_CrateHead",si,"crate renew")
	  if(MsgMissCr==1) then
			AddMsg("errorpk",si,"crate renew in pack:"..tostring(packcnt))
	  end
	end

	crlive[si]=packcnt
	
	if(crtm[si]~=nil) then
		tl=basetime_lnxms-crtm[si]
		--print ("cr bs diff",tl)
		if (tl<=0 or tl>CrTmLim_Ms) then
			s=string.format("crate basetime diff: %d  cur: %s  last: %s",tl,basetime_str,crtms[si])
			AddMsg("errorpk",si,s)
		end
	end
	
	if (basetime_lnxms>0 ) then
		crtm[si]=basetime_lnxms
		crtms[si]=basetime_str
	end
	
	--print (json.encode(crtm))
	
	--print("Add_CrateHead",si,crlive[si],basetime_lnxms)
end


--- from GO ---
--dgname= testl,errl,modgood,....
-- _start_ - MUST first!
function Add_ModDiag2(inputid,dgname,val)
	--print("add md",inputid,dgname,val)

	s=tostring(inputid)

	--get current val
	cval=nil
	if modsdiag[s]~=nil then
		cval=modsdiag[s][dgname]
		--print("cval",s,cval)
		--print("cavl",modsdiag[s]["testl"])
	end

	--clear if start new 
	--if dgname == "_start_" then 
	--	modsdiag[tostring(inputid)]={}
	--	return 
	--end	

	local chgfl=0


	---
	if(modsdiag[s]==nil) then
		modsdiag[s]={}
	end
	modsdiag[s][dgname]=val

	-- test for change ---
	v=modsdiag[s][dgname]
	if cval~=nil then
		--print("cc",cval,val)
		if cval~=v then
			--print("md",cval,val)
			AddDiagChg(inputid,dgname,cval,v)		
			chgfl=1
		end
	end




	--modsdiag ["17893120"]["testl"]=-1   --DEBUG
	--modsdiag ["17893120"]["cntrl_1"]=1   --DEBUG
	--modsdiag ["17957632"]["testl"] = -1 

end


function AddDiagChg(iid,dgname,vprev,vcur)
--print ("AddDiagChg")
if #diagchg>100000 then return end
diagchg[#diagchg+1]={["iid"]=iid,["dgname"]=dgname,["vprev"]=vprev,["vcur"]=vcur}
--print ("AddDiagChg",iid,dgname,vprev,vcur)

s = string.format("%s: prev=%d cur=%d",dgname,vprev,vcur)
AddMsg("diagchg",iid,s)

end


--- from GO ---
function TestDt()
--id="17893120"
 --print ("testdt",id,modsdiag[id]["testl"],modsdiag[id]["errl"])
 --AddMsg("test",0,"teststring")
end

------- test

AddMsg("diagchg",32,"test")





